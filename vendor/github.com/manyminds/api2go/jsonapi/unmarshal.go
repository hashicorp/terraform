package jsonapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
)

// The UnmarshalIdentifier interface must be implemented to set the ID during
// unmarshalling.
type UnmarshalIdentifier interface {
	SetID(string) error
}

// The UnmarshalToOneRelations interface must be implemented to unmarshal
// to-one relations.
type UnmarshalToOneRelations interface {
	SetToOneReferenceID(name, ID string) error
}

// The UnmarshalToManyRelations interface must be implemented to unmarshal
// to-many relations.
type UnmarshalToManyRelations interface {
	SetToManyReferenceIDs(name string, IDs []string) error
}

// The EditToManyRelations interface can be optionally implemented to add and
// delete to-many relationships on a already unmarshalled struct. These methods
// are used by our API for the to-many relationship update routes.
//
// There are 3 HTTP Methods to edit to-many relations:
//
//	PATCH /v1/posts/1/comments
//	Content-Type: application/vnd.api+json
//	Accept: application/vnd.api+json
//
//	{
//	  "data": [
//		{ "type": "comments", "id": "2" },
//		{ "type": "comments", "id": "3" }
//	  ]
//	}
//
// This replaces all of the comments that belong to post with ID 1 and the
// SetToManyReferenceIDs method will be called.
//
//	POST /v1/posts/1/comments
//	Content-Type: application/vnd.api+json
//	Accept: application/vnd.api+json
//
//	{
//	  "data": [
//		{ "type": "comments", "id": "123" }
//	  ]
//	}
//
// Adds a new comment to the post with ID 1.
// The AddToManyIDs method will be called.
//
//	DELETE /v1/posts/1/comments
//	Content-Type: application/vnd.api+json
//	Accept: application/vnd.api+json
//
//	{
//	  "data": [
//		{ "type": "comments", "id": "12" },
//		{ "type": "comments", "id": "13" }
//	  ]
//	}
//
// Deletes comments that belong to post with ID 1.
// The DeleteToManyIDs method will be called.
type EditToManyRelations interface {
	AddToManyIDs(name string, IDs []string) error
	DeleteToManyIDs(name string, IDs []string) error
}

// Unmarshal parses a JSON API compatible JSON and populates the target which
// must implement the `UnmarshalIdentifier` interface.
func Unmarshal(data []byte, target interface{}) error {
	if target == nil {
		return errors.New("target must not be nil")
	}

	if reflect.TypeOf(target).Kind() != reflect.Ptr {
		return errors.New("target must be a ptr")
	}

	ctx := &Document{}

	err := json.Unmarshal(data, ctx)
	if err != nil {
		return err
	}

	if ctx.Data == nil {
		return errors.New(`Source JSON is empty and has no "attributes" payload object`)
	}

	if ctx.Data.DataObject != nil {
		return setDataIntoTarget(ctx.Data.DataObject, target)
	}

	if ctx.Data.DataArray != nil {
		targetSlice := reflect.TypeOf(target).Elem()
		if targetSlice.Kind() != reflect.Slice {
			return fmt.Errorf("Cannot unmarshal array to struct target %s", targetSlice)
		}
		targetType := targetSlice.Elem()
		targetPointer := reflect.ValueOf(target)
		targetValue := targetPointer.Elem()

		for _, record := range ctx.Data.DataArray {
			// check if there already is an entry with the same id in target slice,
			// otherwise create a new target and append
			var targetRecord, emptyValue reflect.Value
			for i := 0; i < targetValue.Len(); i++ {
				marshalCasted, ok := targetValue.Index(i).Interface().(MarshalIdentifier)
				if !ok {
					return errors.New("existing structs must implement interface MarshalIdentifier")
				}
				if record.ID == marshalCasted.GetID() {
					targetRecord = targetValue.Index(i).Addr()
					break
				}
			}

			if targetRecord == emptyValue || targetRecord.IsNil() {
				targetRecord = reflect.New(targetType)
				err := setDataIntoTarget(&record, targetRecord.Interface())
				if err != nil {
					return err
				}
				targetValue = reflect.Append(targetValue, targetRecord.Elem())
			} else {
				err := setDataIntoTarget(&record, targetRecord.Interface())
				if err != nil {
					return err
				}
			}
		}

		targetPointer.Elem().Set(targetValue)
	}

	return nil
}

func setDataIntoTarget(data *Data, target interface{}) error {
	castedTarget, ok := target.(UnmarshalIdentifier)
	if !ok {
		return errors.New("target must implement UnmarshalIdentifier interface")
	}

	if data.Type == "" {
		return errors.New("invalid record, no type was specified")
	}

	err := checkType(data.Type, castedTarget)
	if err != nil {
		return err
	}

	if data.Attributes != nil {
		err = json.Unmarshal(data.Attributes, castedTarget)
		if err != nil {
			return err
		}
	}

	if err := castedTarget.SetID(data.ID); err != nil {
		return err
	}

	return setRelationshipIDs(data.Relationships, castedTarget)
}

// extracts all found relationships and set's them via SetToOneReferenceID or
// SetToManyReferenceIDs
func setRelationshipIDs(relationships map[string]Relationship, target UnmarshalIdentifier) error {
	for name, rel := range relationships {
		// if Data is nil, it means that we have an empty toOne relationship
		if rel.Data == nil {
			castedToOne, ok := target.(UnmarshalToOneRelations)
			if !ok {
				return fmt.Errorf("struct %s does not implement UnmarshalToOneRelations", reflect.TypeOf(target))
			}

			castedToOne.SetToOneReferenceID(name, "")
			break
		}

		// valid toOne case
		if rel.Data.DataObject != nil {
			castedToOne, ok := target.(UnmarshalToOneRelations)
			if !ok {
				return fmt.Errorf("struct %s does not implement UnmarshalToOneRelations", reflect.TypeOf(target))
			}
			err := castedToOne.SetToOneReferenceID(name, rel.Data.DataObject.ID)
			if err != nil {
				return err
			}
		}

		// valid toMany case
		if rel.Data.DataArray != nil {
			castedToMany, ok := target.(UnmarshalToManyRelations)
			if !ok {
				return fmt.Errorf("struct %s does not implement UnmarshalToManyRelations", reflect.TypeOf(target))
			}
			IDs := make([]string, len(rel.Data.DataArray))
			for index, relData := range rel.Data.DataArray {
				IDs[index] = relData.ID
			}
			err := castedToMany.SetToManyReferenceIDs(name, IDs)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func checkType(incomingType string, target UnmarshalIdentifier) error {
	actualType := getStructType(target)
	if incomingType != actualType {
		return fmt.Errorf("Type %s in JSON does not match target struct type %s", incomingType, actualType)
	}

	return nil
}
