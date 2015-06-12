package aws

import (
    "fmt"
    "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsIamPolicyAttach() *schema.Resource {
    return &schema.Resource{
        Create: resourceAwsIamPolicyAttachCreate,
        Read: resourceAwsIamPolicyAttachRead,
        Update: resourceAwsIamPolicyAttachUpdate,
        Delete: resourceAwsIamPolicyAttachDelete,

        Schema: map[]*schema.Schema{
            "name":  &schema.Schema{
                Type: schema.TypeString,
                Required: true,
                ForceNew: true,
            },
            "users": &schema.Schema{
                Type:   schema.TypeSet,
                Optional: true,
                Elem: &schema.Schema{Type: schema.TypeString},
            },
            "roles": &schema.Schema{
                Type:   schema.TypeSet,
                Optional: true,
                Elem: &schema.Schema{Type: schema.TypeString},
            },
            "groups": &schema.Schema{
                Type:   schema.TypeSet,
                Optional: true,
                Elem: &schema.Schema{Type: schema.TypeString},
            },
            "arn": &schema.Schema{
                Type: schema.TypeString,
                Required: true,
            },
        },
    }
}

func resourceAwsIamPolicyAttachCreate(d *schema.ResourceData, meta interface{}) error {
    conn := meta.(*AWSClient).iamconn

    name := d.Get("name").(string)
    arn := d.Get("arn").(string)
    users := expandStringList(d.Get("users").(*schema.Set).List())
    roles := expandStringList(d.Get("roles").(*schema.Set).List())
    groups := expandStringList(d.Get("groups").(*schema.Set).List())

    if users == nil && roles == nil && groups == nil {
        return fmt.Errorf("[WARN] No Users, Roles, or Groups specified for %s", d.Get("name").(string))
    }
    else {
        var userErr, roleErr, groupErr error
        if users != nil {
            userErr = attachPolicyToUsers(conn, users, arn)
        }
        if roles != nil {
            roleErr = attachPolicyToRoles(conn, roles, arn)
        }
        if groups != nil {
            groupErr = attachPolicyToGroups(conn, groups, arn)
        }
        if userErr != nil || roleErr != nil || groupErr != nil {
            return fmt.Errorf("[WARN] Error attaching policy with IAM Policy Attach (%s), error:\n users - %v\n roles - %v\n groups - %v", name, userErr, roleErr, groupErr)
        }
    }
    return resourceAwsIamPolicyAttachRead(d, meta)
}
func resourceAwsIamPolicyAttachRead(d *schema.ResourceData, meta interface{}) error {
    conn := meta.(*AWSClient).iamconn
    users := expandStringList(d.Get("users").(*schema.Set).List())
    roles := expandStringList(d.Get("roles").(*schema.Set).List())
    groups := expandStringList(d.Get("groups").(*schema.Set).List())


    return nil
}
func resourceAwsIamPolicyAttachUpdate(d *schema.ResourceData, meta interface{}) error {
    conn := meta.(*AWSClient).iamconn

    return nil
}
func resourceAwsIamPolicyAttachDelete(d *schema.ResourceData, meta interface{}) error {
    conn := meta.(*AWSClient).iamconn
    name := d.Get("name").(string)
    arn := d.Id()
    users := expandStringList(d.Get("users").(*schema.Set).List())
    roles := expandStringList(d.Get("roles").(*schema.Set).List())
    groups := expandStringList(d.Get("groups").(*schema.Set).List())
    
    var userErr, roleErr, groupErr error
    if users != nil {
        userErr = detachPolicyFromUsers(conn, users, arn)
    }
    if roles != nil {
        roleErr = detachPolicyFromRoles(conn, roles, arn)
    }
    if groups != nil {
        groupErr = detachPolicyFromGroups(conn, groups, arn)
    }
    if userErr != nil || roleErr != nil || groupErr != nil {
        return fmt.Errorf("Error detaching policy with IAM Policy Attach (%s), error:\n users - %v\n roles - %v\ groups - %v", name, userErr, roleErr, groupErr)
    }
    return nil
}
func attachPolicyToUsers (conn *iam.IAM, users []*string, arn string) {
    for _, u := range users {
        _, err := conn.AttachGroupPolicy(&iam.AttachGroupPolicy{
            GroupName: u,
            PolicyArn: aws.String(arn),
        })
        if err != nil {
            return err
        }
    }
    return nil
}
func attachPolicyToRoles (conn *iam.IAM, roles []*string, arn string) {
    for _, r := range roles {
        _, err := conn.AttachRolePolicy(&iam.AttachRolePolicy{
            RoleName: u,
            PolicyArn: aws.String(arn),
        })
        if err != nil {
            return err
        }
    }
    return nil

}
func attachPolicyToGroups (conn *iam.IAM, groups []*string, arn string) {
    for _, g := range groups {
        _, err := conn.AttachGroupPolicy(&iam.AttachGroupPolicy{
            GroupName: g,
            PolicyArn: aws.String(arn),
        })
        if err != nil {
            return err
        }
    }
    return nil
}
func detachPolicyFromUsers(conn *iam.IAM, users []*string, arn string) {
    for _, u := range users {
        _, err := conn.DetachUserPolicy(&iam.DetachUserPolicy{
            UserName: u,
            PolicyArn: aws.String(arn),
        }
        if err != nil {
            return err
        }
    }
    return nil
}
func detachPolicyFromRoles(conn *iam.IAM, roles []*string, arn string) {
    for _, r := range roles {
        _, err := conn.DetachRolePolicy(&iam.DetachRolePolicy{
            RoleName: r,
            PolicyArn: aws.String(arn),
        }
        if err != nil {
            return err
        }
    }
    return nil
}
func detachPolicyFromGroups(conn *iam.IAM, groups []*string, arn string) {
    for _, g := range groups {
        _, err := conn.DetachGroupPolicy(&iam.DetachGroupPolicy{
            GroupName: g,
            PolicyArn: aws.String(arn),
        }
        if err != nil {
            return err
        }
    }
    return nil
}
