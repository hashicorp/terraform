package aws

import (
    "bytes"
    "fmt"
    "log"

    "github.com/awslabs/aws-sdk-go/aws"
    "github.com/awslabs/aws-sdk-go/service/autoscaling"
    "github.com/hashicorp/terraform/helper/hashcode"
    "github.com/hashicorp/terraform/helper/schema"
)

func autoscalingScalingPolicySchema() *schema.Schema {
    return &schema.Schema{
        Type: schema.TypeSet,
        Optional: true,
        Computed: true,
        Elem: &schema.Resource{
            Schema: map[string]*schema.Schema{
                "adjustment_type": &schema.Schema{
                    Type: schema.TypeString,
                    Required: true,
                },
                "auto_scaling_group_name": &schema.Schema{
                    Type: schema.TypeString,
                },
                "cooldown": &schema.Schema{
                    Type: schema.TypeInt,
                    Optional: true,
                },
                "min_adjustment_step": &schema.Schema{
                    Type: schema.TypeInt,
                    Optional: true,
                },
                "policy_arn": &schema.Schema{
                    Type: schema.TypeString,
                    Computed: true,
                },
                "policy_name": &schema.Schema{
                    Type: schema.TypeString,
                    Required: true,
                },
                "scaling_adjustment": &schema.Schema{
                    Type: schema.TypeInt,
                    Required: true,
                },
            },
        },
        Set: resourceAwsAutoscalingGroupScalingPolicyHash,
    }
}

func resourceAwsAutoscalingGroupScalingPolicyHash(v interface{}) int {
    var buf bytes.Buffer
    m := v.(map[string]interface{})
    buf.WriteString(fmt.Sprintf("%s-", m["policy_name"].(string)))
    buf.WriteString(fmt.Sprintf("%d-", m["scaling_adjustment"].(int)))
    buf.WriteString(fmt.Sprintf("%s-", m["adjustment_type"].(string)))
    buf.WriteString(fmt.Sprintf("%d-", m["cooldown"].(int)))
    buf.WriteString(fmt.Sprintf("%d-", m["min_adjustment_step"].(int)))

    return hashcode.String(buf.String())
}

func setAutoscalingScalingPolicies(conn *autoscaling.AutoScaling, d *schema.ResourceData) error {
    if d.HasChange("scaling_policy") {
        o, n := d.GetChange("scaling_policy")
        ors := o.(*schema.Set).Difference(n.(*schema.Set))
        nrs := n.(*schema.Set).Difference(o.(*schema.Set))

        log.Printf("[DEBUG] o %#v", lRaw)
        log.Printf("[DEBUG] n %#v", lRaw)

        return nil
        //log.Printf("[DEBUG] Put Scaling Policy %#v", lRaw)
        //log.Printf("[DEBUG] Put Scaling Policy %#v", lRaw)

        // Loop through and delete old scaling policy
        for _, policy := range ors.List() {
            m := policy.(map[string]interface{})

            params := autoscaling.DeletePolicyInput{
                PolicyName:             aws.String(m["policy_name"]),
                AutoScalingGroupName:   aws.String(d.Get("name")),
            }

            resp, err := conn.DeletePolicy(&params)
            if err != nil {
                return err
            }
        }

        for _, policy := range nrs.List() {
            m := policy.(map[string]interface{})
            params := autoscaling.PutScalingPolicyInput{
                AutoScalingGroupName:   aws.String(d.Id()),
                PolicyName:             aws.String(m["policy_name"]),
            }

            if v, ok := data["adjustment_type"]; ok {
                p.AdjustmentType = aws.String(v.(string))
            }

            if v, ok := data["scaling_adjustment"]; ok {
                p.ScalingAdjustment = aws.Integer(v.(int))
            }

            if v, ok := data["min_adjustment_step"]; ok && v != 0 {
                p.MinAdjustmentStep = aws.Integer(v.(int))
            }

            if v, ok := data["cooldown"]; ok {
                p.Cooldown = aws.Integer(v.(int))
            }

            resp, err := conn.PutScalingPolicy(&params)
        }

        raw := d.Get("scaling_policy").(*schema.Set)
        for _, lRaw := range raw.List(){
            data := lRaw.(map[string]interface{})
            p := autoscaling.PutScalingPolicyInput{
                AutoScalingGroupName : aws.String(d.Get("name").(string)),
                PolicyName           : aws.String(data["policy_name"].(string)),
            }

            log.Printf("[DEBUG] Put Scaling Policy %#v", lRaw)
            policyResult, err := conn.PutScalingPolicy(&p)
            if err != nil {
                return err
            }
            lRaw.Set("policy_arn", policyResult.PolicyARN)
        }
    }
    return nil
}
