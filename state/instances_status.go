package state

import (
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"sync"
)

const (
	// Meta ID is transitional id. It doesn't make sense for terraform core logic
	// and this used only for determining instances without own id.
	MetaIdKey    string = "metaId"
	MetaIdPrefix string = "meta-id-"
)

var instancesStatusLogger *InstancesStatusLogger

func init() {
	// Initialize InstancesStatusLogger as singleton.
	instancesStatusLogger = &InstancesStatusLogger{
		recoveryLog:     map[string]Instance{},
		lostResourceLog: map[string]LostInstance{},
	}
}

// Object contains recovery information
// and provides methods to synchronize information between terraform and remote backend.
type InstancesStatusLogger struct {
	recoveryLog     map[string]Instance
	lostResourceLog map[string]LostInstance
	lock            sync.Mutex
}

type Instance struct {
	Id      string `json:"id"`
	Address string `json:"address"`
}
type LostInstance struct {
	Id string

	// ModulePath is the complete path of the module containing this
	// instance.
	ModulePath []string

	// Type is the resource type of this instance
	Type string

	Attributes []string
}

func GetGlobalInstancesStatusLogger() *InstancesStatusLogger {
	return instancesStatusLogger
}

func (s *InstancesStatusLogger) Add(id string, info *terraform.InstanceInfo, instState *terraform.InstanceState, writer RecoveryLogWriter) {
	fmt.Println("Add Instance status")
	s.lock.Lock()
	defer s.lock.Unlock()

	if metaId, ok := instState.Meta[MetaIdKey]; ok {
		metaIdString := metaId.(string)
		if _, found := s.lostResourceLog[metaIdString]; found {
			delete(s.lostResourceLog, metaIdString)
			s.writeLostResourceLog(writer)
		}
	}

	s.recoveryLog[id] = Instance{Id: id, Address: info.Id}
	s.writeRecoveryLog(writer)
}

// Set instance as lost resource mark it unique meta id
// and  write additional info about instance to remote state.
func (s *InstancesStatusLogger) SetLostResource(
	instInfo *terraform.InstanceInfo,
	instState *terraform.InstanceState,
	diff *terraform.InstanceDiff,
	writer RecoveryLogWriter) {

	s.lock.Lock()
	defer s.lock.Unlock()

	metaId := resource.PrefixedUniqueId(MetaIdPrefix)
	instState.Ephemeral.MetaId = metaId

	s.lostResourceLog[metaId] = LostInstance{
		Id:         instInfo.Id,
		Type:       instInfo.Type,
		ModulePath: instInfo.ModulePath,
		Attributes: formatAttributes(diff),
	}

	fmt.Printf("Add instance to Lost Resources log with metaID: %s\n", metaId)
	s.writeLostResourceLog(writer)
}

// Remove instance from recovery log
// and lost resource log if instance has meta id.
func (s *InstancesStatusLogger) Remove(instState *terraform.InstanceState, info *terraform.InstanceInfo, writer RecoveryLogWriter) {
	fmt.Println("Remove Instance status...")
	s.lock.Lock()
	defer s.lock.Unlock()

	if instState != nil && instState.Ephemeral.MetaId != "" {
		fmt.Printf("Instance status found by metaID into lost resource log. Remove instance from Lost Resources log. metaId: %s", instState.Ephemeral.MetaId)
		if _, found := s.lostResourceLog[instState.Ephemeral.MetaId]; found {
			delete(s.lostResourceLog, instState.Ephemeral.MetaId)
			s.writeLostResourceLog(writer)
		}
	}

	_, ok := s.recoveryLog[instState.ID]
	if ok {
		delete(s.recoveryLog, instState.ID)
		s.writeRecoveryLog(writer)
	}
}

func (s *InstancesStatusLogger) writeRecoveryLog(writer RecoveryLogWriter) {
	if data, err := json.Marshal(s.recoveryLog); err == nil {
		err = writer.WriteRecoveryLog(data)
		if err != nil {
			fmt.Printf("Error into WriteRecoveryLog: %v \n", err)
		}
	}
}

func (s *InstancesStatusLogger) writeLostResourceLog(writer RecoveryLogWriter) {
	if data, err := json.Marshal(s.lostResourceLog); err == nil {
		err = writer.WriteLostResourceLog(data)
		if err != nil {
			fmt.Printf("Error into WriteLostResourceLog: %v \n", err)
		}
	}
}

func formatAttributes(diff *terraform.InstanceDiff) []string {
	attributes := []string{}

	for k, v := range diff.Attributes {
		attributes = append(attributes, fmt.Sprintf("AttributeKey: %s {%s}", k, v.GoString()))
	}
	return attributes
}
