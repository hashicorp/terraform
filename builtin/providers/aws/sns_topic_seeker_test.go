package aws

import (
	"fmt"
	"testing"
	"time"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/service/sns"
)

// tokenMap mimics paging of topic results.
var tokenMap = map[string]*sns.ListTopicsOutput{
	"tokenA": &sns.ListTopicsOutput{
		Topics: []*sns.Topic{
			&sns.Topic{
				TopicARN: aws.String("arn:aws:sns:us-east-1:123456789012:foo"),
			},
		},
		NextToken: aws.String("tokenB"),
	},
	"tokenB": &sns.ListTopicsOutput{
		Topics: []*sns.Topic{
			&sns.Topic{
				TopicARN: aws.String("arn:aws:sns:us-east-1:123456789012:bar"),
			},
		},
		NextToken: aws.String("tokenC"),
	},
	"tokenC": &sns.ListTopicsOutput{
		Topics: []*sns.Topic{
			&sns.Topic{
				TopicARN: aws.String("arn:aws:sns:us-east-1:123456789012:baz"),
			},
		},
	},
}

// ltErr mimics a ListTopics error
var ltErr = fmt.Errorf("Got an error listing topics!")

// snsConnMock implements topicLister, so we can mock result paging.
type snsConnMock struct {
	topics map[string]*sns.ListTopicsOutput
	err    error
}

// ListTopics returns paged results out of the tokenMap.
func (s *snsConnMock) ListTopics(input *sns.ListTopicsInput) (*sns.ListTopicsOutput, error) {
	var emptyResult sns.ListTopicsOutput

	switch {
	case s.topics == nil:
		// return an empty result
		return &emptyResult, nil
	case s.err != nil:
		// return a mocked response error
		return nil, s.err
	case input.NextToken == nil:
		// we're the first call, return tokenA
		return tokenMap["tokenA"], nil
	default:
		// we're somewhere in the pagination.
		return tokenMap[*input.NextToken], nil
	}
}

func Test_snsTopicSeeker_emit(t *testing.T) {
	s := &snsTopicSeeker{
		arns: make(chan string),
	}
	topic := &sns.Topic{
		TopicARN: aws.String("arn:aws:sns:us-east-1:123456789012:foo"),
	}

	go s.emit(topic)

	// arns is a synchronous channel, so we can't simply block on a read if
	// we're trying to test something sent to it. Instead, use a timeout.
	timeout := make(chan bool, 1)
	go func() {
		time.Sleep(10 * time.Second)
		timeout <- true
	}()

	select {
	case <-s.arns:
		return
	case <-timeout:
		t.Fatalf("emit didn't send a topic to the 'arns' channel!")
	}
}

func Test_snsTopicSeeker_errorf(t *testing.T) {
	s := &snsTopicSeeker{
		errc: make(chan error, 1),
	}

	// errc is buffered. No need for a goroutine here.
	s.errorf(fmt.Errorf("This is an error"))

	select {
	case <-s.errc:
		return
	default:
	}
	t.Fatal("errorf didn't send an error to the 'errc' channel!")
}

func Test_snsTopicSeeker(t *testing.T) {
	for _, ts := range []struct {
		name      string
		lister    *snsConnMock
		shouldErr bool
		wantedLen int
	}{
		{"paginated response", &snsConnMock{topics: tokenMap}, false, 3},
		{"error response", &snsConnMock{topics: tokenMap, err: ltErr}, true, 0},
		{"no topics response", &snsConnMock{}, false, 0},
	} {
		// build the seeker
		s := &snsTopicSeeker{
			lister: ts.lister,
			arns:   make(chan string),
			errc:   make(chan error, 1),
		}

		// run the seeker
		go s.run()

		// walk our response
		var walkedArns []string
		for arn := range s.arns {
			walkedArns = append(walkedArns, arn)
		}

		if len(walkedArns) != ts.wantedLen {
			t.Fatalf("%s: expected %d ARNs; got %d", ts.name, ts.wantedLen, len(walkedArns))
		}

		err := <-s.errc
		if ts.shouldErr && err == nil {
			t.Fatalf("%s: expected error; didn't get one", ts.name)
		}
		if !ts.shouldErr && err != nil {
			t.Fatalf("%s: expected no error; got one: %s", ts.name, err)
		}
	}
}
