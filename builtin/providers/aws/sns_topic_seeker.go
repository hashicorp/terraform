package aws

import (
	"github.com/awslabs/aws-sdk-go/service/sns"
)

// seekSnsTopic starts a topic seeker and reads out the results, looking for
// a particular ARN.
func seekSnsTopic(soughtArn string, snsconn snsTopicLister) (string, error) {
	s := &snsTopicSeeker{
		lister: snsconn,
		arns:   make(chan string),
		errc:   make(chan error, 1),
	}

	// launch the seeker
	go s.run()

	for arn := range s.arns {
		if arn == soughtArn {
			return arn, nil
		}
	}
	if err := <-s.errc; err != nil {
		return "", err
	}

	// We never found the ARN.
	return "", nil
}

// snsTopicLister implements ListTopics. It exists so we can mock out an SNS
// connection for the seeker in testing.
type snsTopicLister interface {
	ListTopics(*sns.ListTopicsInput) (*sns.ListTopicsOutput, error)
}

// seekerStateFn represents the state of the pager as a function that returns
// the next state.
type snsTopicSeekerStateFn func(*snsTopicSeeker) snsTopicSeekerStateFn

// snsTopicSeeker holds the state of our SNS API scanner.
type snsTopicSeeker struct {
	lister   snsTopicLister        // an SNS connection or mock
	token    *string               // the token for the list topics request
	respList []*sns.Topic          // the list of topics in the AWS response
	state    snsTopicSeekerStateFn // the next state function
	arns     chan string           // channel of topic ARNs
	errc     chan error            // buffered error channel
}

// run the seeker
func (s *snsTopicSeeker) run() {
	for s.state = listTopics; s.state != nil; {
		s.state = s.state(s)
	}
	close(s.arns)
	close(s.errc)
}

// emit a topic's ARN onto the arns channel
func (s *snsTopicSeeker) emit(topic *sns.Topic) {
	s.arns <- *topic.TopicARN
}

// errorf sends an error on the error channel and returns nil, stopping the
// seeker.
func (s *snsTopicSeeker) errorf(err error) snsTopicSeekerStateFn {
	s.errc <- err
	return nil
}

// listTopics calls AWS for topics
func listTopics(s *snsTopicSeeker) snsTopicSeekerStateFn {
	resp, err := s.lister.ListTopics(&sns.ListTopicsInput{
		NextToken: s.token,
	})
	switch {
	case err != nil:
		return s.errorf(err)
	case len(resp.Topics) == 0:
		// We've no topics in SNS at all.
		return nil
	default:
		s.respList = resp.Topics
		s.token = resp.NextToken
		return yieldTopic
	}
}

// yieldTopics shifts the seeker's topic list and emits the first item.
func yieldTopic(s *snsTopicSeeker) snsTopicSeekerStateFn {
	topic, remaining := s.respList[0], s.respList[1:]
	s.emit(topic)
	s.respList = remaining
	switch {
	case len(s.respList) > 0:
		return yieldTopic
	case s.token != nil:
		return listTopics
	default:
		return nil
	}
}
