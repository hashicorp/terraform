package aws

import (
	"encoding/json"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

type ecrPolicy struct {
	Version    string                `json:",omitempty"`
	ID         string                `json:",omitempty"`
	Statements []*ecrPolicyStatement `json:"Statement"`
}

type ecrPolicyStatement struct {
	Sid           string
	Effect        string      `json:",omitempty"`
	Actions       interface{} `json:"Action,omitempty"`
	NotActions    interface{} `json:"NotAction,omitempty"`
	Resources     interface{} `json:"Resource,omitempty"`
	NotResources  interface{} `json:"NotResource,omitempty"`
	Principals    interface{} `json:"Principal,omitempty"`
	NotPrincipals interface{} `json:"NotPrincipal,omitempty"`
	Conditions    interface{} `json:"Condition,omitempty"`
}

func resourceAwsEcrRepositoryPolicyStatement() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsEcrRepositoryPolicyStatementCreate,
		Read:   resourceAwsEcrRepositoryPolicyStatementRead,
		Update: resourceAwsEcrRepositoryPolicyStatementUpdate,
		Delete: resourceAwsEcrRepositoryPolicyStatementDelete,

		Schema: map[string]*schema.Schema{
			"sid": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{"sid_prefix"},
			},

			"sid_prefix": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"repository": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"statement": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"registry_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsEcrRepositoryPolicyStatementCreateOrUpdate(d *schema.ResourceData, meta interface{}) error {
	var sid string

	conn := meta.(*AWSClient).ecrconn
	registryID := d.Get("registry_id").(string)
	repository := d.Get("repository").(string)
	stmtText := d.Get("statement").(string)

	if v, ok := d.GetOk("sid"); ok {
		sid = v.(string)
	} else if v, ok := d.GetOk("sid_prefix"); ok {
		sid = resource.PrefixedUniqueId(v.(string))
	} else {
		sid = resource.UniqueId()
	}

	awsMutexKV.Lock(repository)
	defer awsMutexKV.Unlock(repository)

	_, policy, err := readOrBuildEcrPolicy(conn, registryID, repository)
	if err != nil {
		return err
	}

	index, found := findEcrPolicyStatement(policy, sid)

	statement := &ecrPolicyStatement{}
	if found {
		statement = policy.Statements[index]
	} else {
		policy.Statements = append(policy.Statements, statement)
	}

	if err := json.Unmarshal([]byte(stmtText), statement); err != nil {
		return err
	}

	// Force Sid to be the one defined in the resource
	statement.Sid = sid

	repositoryPolicy, err := resourceAwsEcrRepositoryPolicyStatementUpdateWith(d, meta, policy)
	if err != nil {
		return err
	}

	d.SetId(sid)
	d.Set("sid", sid)
	d.Set("repository", *repositoryPolicy.RepositoryName)
	d.Set("registry_id", repositoryPolicy.RegistryId)

	return nil
}

func resourceAwsEcrRepositoryPolicyStatementCreate(d *schema.ResourceData, meta interface{}) error {
	return resourceAwsEcrRepositoryPolicyStatementCreateOrUpdate(d, meta)
}

func resourceAwsEcrRepositoryPolicyStatementRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ecrconn
	sid := d.Get("sid").(string)
	registryID := d.Get("registry_id").(string)
	repository := d.Get("repository").(string)

	awsMutexKV.Lock(repository)
	defer awsMutexKV.Unlock(repository)

	log.Printf("[DEBUG] Reading repository policy statement %s", d.Id())

	repositoryPolicy, policy, err := readOrBuildEcrPolicy(conn, registryID, repository)
	if err != nil {
		return err
	}

	index, found := findEcrPolicyStatement(policy, sid)
	if !found {
		d.SetId("")
		return nil
	}

	statement := policy.Statements[index]

	d.SetId(statement.Sid)
	d.Set("sid", statement.Sid)
	d.Set("registry_id", repositoryPolicy.RegistryId)
	d.Set("repository", *repositoryPolicy.RepositoryName)

	return nil
}

func resourceAwsEcrRepositoryPolicyStatementUpdate(d *schema.ResourceData, meta interface{}) error {
	return resourceAwsEcrRepositoryPolicyStatementCreateOrUpdate(d, meta)
}

func resourceAwsEcrRepositoryPolicyStatementDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ecrconn
	sid := d.Get("sid").(string)
	registryID := d.Get("registry_id").(string)
	repository := d.Get("repository").(string)

	awsMutexKV.Lock(repository)
	defer awsMutexKV.Unlock(repository)

	log.Printf("[DEBUG] Reading repository policy statement %s", d.Id())

	_, policy, err := readOrBuildEcrPolicy(conn, registryID, repository)
	if err != nil {
		return err
	}

	index, found := findEcrPolicyStatement(policy, sid)
	if !found {
		d.SetId("")
		return nil
	}

	policy.Statements = append(policy.Statements[:index], policy.Statements[index+1:]...)

	_, err = resourceAwsEcrRepositoryPolicyStatementUpdateWith(d, meta, policy)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] repository policy statement %s deleted.", d.Id())

	return nil
}

func resourceAwsEcrRepositoryPolicyStatementUpdateWith(d *schema.ResourceData, meta interface{}, policy *ecrPolicy) (*ecr.SetRepositoryPolicyOutput, error) {
	conn := meta.(*AWSClient).ecrconn
	registryID := d.Get("registry_id").(string)
	repository := d.Get("repository").(string)

	if len(policy.Statements) == 0 {
		input := ecr.DeleteRepositoryPolicyInput{
			RepositoryName: aws.String(repository),
		}

		if registryID != "" {
			input.RegistryId = aws.String(registryID)
		}

		_, err := conn.DeleteRepositoryPolicy(&input)

		if err != nil {
			return nil, err
		}

		return nil, nil
	}

	policyText, err := json.Marshal(policy)
	if err != nil {
		return nil, err
	}

	input := ecr.SetRepositoryPolicyInput{
		RepositoryName: aws.String(repository),
		PolicyText:     aws.String(string(policyText)),
	}

	if registryID != "" {
		input.RegistryId = aws.String(registryID)
	}

	out, err := conn.SetRepositoryPolicy(&input)

	if err != nil {
		return nil, err
	}

	return out, nil
}

func readOrBuildEcrPolicy(conn *ecr.ECR, registryID string, repositoryName string) (*ecr.GetRepositoryPolicyOutput, *ecrPolicy, error) {
	input := &ecr.GetRepositoryPolicyInput{
		RepositoryName: aws.String(repositoryName),
	}

	if registryID != "" {
		input.RegistryId = aws.String(registryID)
	}

	out, err := conn.GetRepositoryPolicy(input)

	if err != nil {
		if ecrerr, ok := err.(awserr.Error); ok {
			switch ecrerr.Code() {
			case "RepositoryNotFoundException", "RepositoryPolicyNotFoundException":
				// Do nothing
			default:
				return nil, nil, err
			}
		} else {
			return nil, nil, err
		}
	}

	policy := &ecrPolicy{}
	if out != nil && out.PolicyText != nil {
		if err := json.Unmarshal([]byte(*out.PolicyText), policy); err != nil {
			return nil, nil, err
		}
	} else {
		policy.Version = "2008-10-17"
	}

	if policy.Statements == nil {
		policy.Statements = make([]*ecrPolicyStatement, 0)
	}

	return out, policy, nil
}

func findEcrPolicyStatement(policy *ecrPolicy, sid string) (int, bool) {
	for i, statement := range policy.Statements {
		if statement.Sid == sid {
			return i, true
		}
	}

	return 0, false
}
