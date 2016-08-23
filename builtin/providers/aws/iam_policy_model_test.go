package aws

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestNormalizePolicyDocument(t *testing.T) {
	actual := normalizePolicyDocument(testNormalizePolicyDocumentInputJSON)

	var expected bytes.Buffer
	err := json.Compact(&expected, []byte(testNormalizePolicyDocumentExpectedJSON))
	if err != nil {
		t.Fatal(err)
	}

	if actual != expected.String() {
		t.Errorf("Got: %v\nExpected: %v", actual, expected)
	}
}

const testNormalizePolicyDocumentInputJSON = `
{
  "Id": "policy_id",
  "Statement": [
    {
      "Sid": "1",
      "Effect": "Allow",
      "Action": [
        "s3:ListAllMyBuckets",
        "s3:GetBucketLocation"
      ],
      "NotAction": [],
      "Resource": [
        "arn:aws:s3:::*"
      ],
      "NotResource": []
    },
    {
      "Effect": "Allow",
      "Action": "s3:ListBucket",
      "Resource": "arn:aws:s3:::foo",
      "NotPrincipal": {
        "AWS": [
          "arn:blahblah:example"
        ]
      },
      "Condition": {
        "StringLike": {
          "s3:prefix": [
            "home/${aws:username}/",
            "home/",
            ""
          ]
        }
      }
    }
  ],
  "Version": "2012-10-17"
}
`

const testNormalizePolicyDocumentExpectedJSON = `
{
  "Version": "2012-10-17",
  "Id": "policy_id",
  "Statement": [
    {
      "Sid": "1",
      "Effect": "Allow",
      "Action": [
        "s3:GetBucketLocation",
        "s3:ListAllMyBuckets"
      ],
      "Resource": "arn:aws:s3:::*"
    },
    {
      "Effect": "Allow",
      "Action": "s3:ListBucket",
      "Resource": "arn:aws:s3:::foo",
      "NotPrincipal": {
        "AWS": "arn:blahblah:example"
      },
      "Condition": {
        "StringLike": {
          "s3:prefix": [
            "",
            "home/",
            "home/${aws:username}/"
          ]
        }
      }
    }
  ]
}
`
