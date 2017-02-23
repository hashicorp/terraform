/**
 * Copyright 2016 IBM Corp.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

/**
 * AUTOMATICALLY GENERATED CODE - DO NOT MODIFY
 */

package datatypes

// The SoftLayer_Survey data type contains general information relating to a single SoftLayer survey.
type Survey struct {
	Entity

	// A flag indicating if a survey can be taken.
	Active *int `json:"active,omitempty" xmlrpc:"active,omitempty"`

	// The date that a survey had originally started.
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// A survey's id.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// A survey's name or title.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// A count of the questions for a survey.
	QuestionCount *uint `json:"questionCount,omitempty" xmlrpc:"questionCount,omitempty"`

	// The questions for a survey.
	Questions []Survey_Question `json:"questions,omitempty" xmlrpc:"questions,omitempty"`

	// The status of the survey
	Status *Survey_Status `json:"status,omitempty" xmlrpc:"status,omitempty"`

	// The status id of the survey.
	StatusId *int `json:"statusId,omitempty" xmlrpc:"statusId,omitempty"`

	// The type of survey
	Type *Survey_Type `json:"type,omitempty" xmlrpc:"type,omitempty"`

	// The type id of the survey.
	TypeId *int `json:"typeId,omitempty" xmlrpc:"typeId,omitempty"`
}

// The SoftLayer_Survey_Answer data type contains general information relating to a single SoftLayer survey answer.
type Survey_Answer struct {
	Entity

	// A survey answer's answer that a user can response too.
	Answer *string `json:"answer,omitempty" xmlrpc:"answer,omitempty"`

	// A value indicating the order in when a survey answer will be displayed to a user.
	AnswerOrder *int `json:"answerOrder,omitempty" xmlrpc:"answerOrder,omitempty"`

	// A survey answer's Id.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The survey question that this answer belongs to.
	SurveyQuestion *Survey_Question `json:"surveyQuestion,omitempty" xmlrpc:"surveyQuestion,omitempty"`

	// A survey answer's associated [[SoftLayer_Survey_Question|Survey Question]] Id.
	SurveyQuestionId *int `json:"surveyQuestionId,omitempty" xmlrpc:"surveyQuestionId,omitempty"`
}

// The SoftLayer_Survey_Question data type contains general information relating to a single SoftLayer survey question.
type Survey_Question struct {
	Entity

	// A count of the possible answers for a survey question.
	AnswerCount *uint `json:"answerCount,omitempty" xmlrpc:"answerCount,omitempty"`

	// The possible answers for a survey question.
	Answers []Survey_Answer `json:"answers,omitempty" xmlrpc:"answers,omitempty"`

	// A survey question's Id.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// A flag indicating that a survey question requires a response.
	IsRequired *int `json:"isRequired,omitempty" xmlrpc:"isRequired,omitempty"`

	// A flag indicating that a survey question can have multiple answers responded to.
	MultiAnswer *int `json:"multiAnswer,omitempty" xmlrpc:"multiAnswer,omitempty"`

	// A survey question's question.
	Question *string `json:"question,omitempty" xmlrpc:"question,omitempty"`

	// A value indicating the order in when a survey question will be asked.
	QuestionOrder *int `json:"questionOrder,omitempty" xmlrpc:"questionOrder,omitempty"`

	// The survey that a question belongs to.
	Survey *Survey `json:"survey,omitempty" xmlrpc:"survey,omitempty"`

	// A survey question's associated [[SoftLayer_Survey|Survey]] Id.
	SurveyId *int `json:"surveyId,omitempty" xmlrpc:"surveyId,omitempty"`
}

// The SoftLayer_Survey_Response data type contains general information relating to a single SoftLayer survey response.
type Survey_Response struct {
	Entity

	// The user typed response for the [[SoftLayer_Survey_Answer|Survey Answer]] that a response is associated with.
	OtherAnswer *string `json:"otherAnswer,omitempty" xmlrpc:"otherAnswer,omitempty"`

	// The survey answer that this response was to.
	SurveyAnswer *Survey_Answer `json:"surveyAnswer,omitempty" xmlrpc:"surveyAnswer,omitempty"`

	// The Id of the [[SoftLayer_Survey_Answer|Survey Answer]] that a response was made for.
	SurveyAnswerId *int `json:"surveyAnswerId,omitempty" xmlrpc:"surveyAnswerId,omitempty"`
}

// The SoftLayer_Survey_Status data type contains survey status information.
type Survey_Status struct {
	Entity

	// Description of a survey status
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// Internal identifier of a survey status
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// Name of a survey status
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// The SoftLayer_Survey_Type data type contains survey type information.
type Survey_Type struct {
	Entity

	// Description of a survey type
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// Internal identifier of a survey type
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// Name of a survey type
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}
