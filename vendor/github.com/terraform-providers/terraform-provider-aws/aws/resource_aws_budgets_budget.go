package aws

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/budgets"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsBudgetsBudget() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"account_id": {
				Type:         schema.TypeString,
				Computed:     true,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validateAwsAccountId,
			},
			"name": {
				Type:          schema.TypeString,
				Computed:      true,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"name_prefix"},
			},
			"name_prefix": {
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
				ForceNew: true,
			},
			"budget_type": {
				Type:     schema.TypeString,
				Required: true,
			},
			"limit_amount": {
				Type:     schema.TypeString,
				Required: true,
			},
			"limit_unit": {
				Type:     schema.TypeString,
				Required: true,
			},
			"cost_types": {
				Type:     schema.TypeList,
				Computed: true,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"include_credit": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
						"include_discount": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
						"include_other_subscription": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
						"include_recurring": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
						"include_refund": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
						"include_subscription": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
						"include_support": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
						"include_tax": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
						"include_upfront": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
						"use_amortized": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
						"use_blended": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
					},
				},
			},
			"time_period_start": {
				Type:     schema.TypeString,
				Required: true,
			},
			"time_period_end": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "2087-06-15_00:00",
			},
			"time_unit": {
				Type:     schema.TypeString,
				Required: true,
			},
			"cost_filters": {
				Type:     schema.TypeMap,
				Optional: true,
				Computed: true,
			},
		},
		Create: resourceAwsBudgetsBudgetCreate,
		Read:   resourceAwsBudgetsBudgetRead,
		Update: resourceAwsBudgetsBudgetUpdate,
		Delete: resourceAwsBudgetsBudgetDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
	}
}

func resourceAwsBudgetsBudgetCreate(d *schema.ResourceData, meta interface{}) error {
	budget, err := expandBudgetsBudgetUnmarshal(d)
	if err != nil {
		return fmt.Errorf("failed unmarshalling budget: %v", err)
	}

	if v, ok := d.GetOk("name"); ok {
		budget.BudgetName = aws.String(v.(string))

	} else if v, ok := d.GetOk("name_prefix"); ok {
		budget.BudgetName = aws.String(resource.PrefixedUniqueId(v.(string)))

	} else {
		budget.BudgetName = aws.String(resource.UniqueId())
	}

	client := meta.(*AWSClient).budgetconn
	var accountID string
	if v, ok := d.GetOk("account_id"); ok {
		accountID = v.(string)
	} else {
		accountID = meta.(*AWSClient).accountid
	}

	_, err = client.CreateBudget(&budgets.CreateBudgetInput{
		AccountId: aws.String(accountID),
		Budget:    budget,
	})
	if err != nil {
		return fmt.Errorf("create budget failed: %v", err)
	}

	d.SetId(fmt.Sprintf("%s:%s", accountID, *budget.BudgetName))
	return resourceAwsBudgetsBudgetRead(d, meta)
}

func resourceAwsBudgetsBudgetRead(d *schema.ResourceData, meta interface{}) error {
	accountID, budgetName, err := decodeBudgetsBudgetID(d.Id())
	if err != nil {
		return err
	}

	client := meta.(*AWSClient).budgetconn
	describeBudgetOutput, err := client.DescribeBudget(&budgets.DescribeBudgetInput{
		BudgetName: aws.String(budgetName),
		AccountId:  aws.String(accountID),
	})
	if isAWSErr(err, budgets.ErrCodeNotFoundException, "") {
		log.Printf("[WARN] Budget %s not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err != nil {
		return fmt.Errorf("describe budget failed: %v", err)
	}

	budget := describeBudgetOutput.Budget
	if budget == nil {
		log.Printf("[WARN] Budget %s not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	d.Set("account_id", accountID)
	d.Set("budget_type", budget.BudgetType)

	if err := d.Set("cost_filters", convertCostFiltersToStringMap(budget.CostFilters)); err != nil {
		return fmt.Errorf("error setting cost_filters: %s", err)
	}

	if err := d.Set("cost_types", flattenBudgetsCostTypes(budget.CostTypes)); err != nil {
		return fmt.Errorf("error setting cost_types: %s %s", err, budget.CostTypes)
	}

	if budget.BudgetLimit != nil {
		d.Set("limit_amount", budget.BudgetLimit.Amount)
		d.Set("limit_unit", budget.BudgetLimit.Unit)
	}

	d.Set("name", budget.BudgetName)

	if budget.TimePeriod != nil {
		d.Set("time_period_end", aws.TimeValue(budget.TimePeriod.End).Format("2006-01-02_15:04"))
		d.Set("time_period_start", aws.TimeValue(budget.TimePeriod.Start).Format("2006-01-02_15:04"))
	}

	d.Set("time_unit", budget.TimeUnit)

	return nil
}

func resourceAwsBudgetsBudgetUpdate(d *schema.ResourceData, meta interface{}) error {
	accountID, _, err := decodeBudgetsBudgetID(d.Id())
	if err != nil {
		return err
	}

	client := meta.(*AWSClient).budgetconn
	budget, err := expandBudgetsBudgetUnmarshal(d)
	if err != nil {
		return fmt.Errorf("could not create budget: %v", err)
	}

	_, err = client.UpdateBudget(&budgets.UpdateBudgetInput{
		AccountId: aws.String(accountID),
		NewBudget: budget,
	})
	if err != nil {
		return fmt.Errorf("update budget failed: %v", err)
	}

	return resourceAwsBudgetsBudgetRead(d, meta)
}

func resourceAwsBudgetsBudgetDelete(d *schema.ResourceData, meta interface{}) error {
	accountID, budgetName, err := decodeBudgetsBudgetID(d.Id())
	if err != nil {
		return err
	}

	client := meta.(*AWSClient).budgetconn
	_, err = client.DeleteBudget(&budgets.DeleteBudgetInput{
		BudgetName: aws.String(budgetName),
		AccountId:  aws.String(accountID),
	})
	if err != nil {
		if isAWSErr(err, budgets.ErrCodeNotFoundException, "") {
			log.Printf("[INFO] budget %s could not be found. skipping delete.", d.Id())
			return nil
		}

		return fmt.Errorf("delete budget failed: %v", err)
	}

	return nil
}

func flattenBudgetsCostTypes(costTypes *budgets.CostTypes) []map[string]interface{} {
	if costTypes == nil {
		return []map[string]interface{}{}
	}

	m := map[string]interface{}{
		"include_credit":             aws.BoolValue(costTypes.IncludeCredit),
		"include_discount":           aws.BoolValue(costTypes.IncludeDiscount),
		"include_other_subscription": aws.BoolValue(costTypes.IncludeOtherSubscription),
		"include_recurring":          aws.BoolValue(costTypes.IncludeRecurring),
		"include_refund":             aws.BoolValue(costTypes.IncludeRefund),
		"include_subscription":       aws.BoolValue(costTypes.IncludeSubscription),
		"include_support":            aws.BoolValue(costTypes.IncludeSupport),
		"include_tax":                aws.BoolValue(costTypes.IncludeTax),
		"include_upfront":            aws.BoolValue(costTypes.IncludeUpfront),
		"use_amortized":              aws.BoolValue(costTypes.UseAmortized),
		"use_blended":                aws.BoolValue(costTypes.UseBlended),
	}
	return []map[string]interface{}{m}
}

func convertCostFiltersToStringMap(costFilters map[string][]*string) map[string]string {
	convertedCostFilters := make(map[string]string)
	for k, v := range costFilters {
		filterValues := make([]string, 0)
		for _, singleFilterValue := range v {
			filterValues = append(filterValues, *singleFilterValue)
		}

		convertedCostFilters[k] = strings.Join(filterValues, ",")
	}

	return convertedCostFilters
}

func expandBudgetsBudgetUnmarshal(d *schema.ResourceData) (*budgets.Budget, error) {
	budgetName := d.Get("name").(string)
	budgetType := d.Get("budget_type").(string)
	budgetLimitAmount := d.Get("limit_amount").(string)
	budgetLimitUnit := d.Get("limit_unit").(string)
	costTypes := expandBudgetsCostTypesUnmarshal(d.Get("cost_types").([]interface{}))
	budgetTimeUnit := d.Get("time_unit").(string)
	budgetCostFilters := make(map[string][]*string)
	for k, v := range d.Get("cost_filters").(map[string]interface{}) {
		filterValue := v.(string)
		budgetCostFilters[k] = append(budgetCostFilters[k], aws.String(filterValue))
	}

	budgetTimePeriodStart, err := time.Parse("2006-01-02_15:04", d.Get("time_period_start").(string))
	if err != nil {
		return nil, fmt.Errorf("failure parsing time: %v", err)
	}

	budgetTimePeriodEnd, err := time.Parse("2006-01-02_15:04", d.Get("time_period_end").(string))
	if err != nil {
		return nil, fmt.Errorf("failure parsing time: %v", err)
	}

	budget := &budgets.Budget{
		BudgetName: aws.String(budgetName),
		BudgetType: aws.String(budgetType),
		BudgetLimit: &budgets.Spend{
			Amount: aws.String(budgetLimitAmount),
			Unit:   aws.String(budgetLimitUnit),
		},
		CostTypes: costTypes,
		TimePeriod: &budgets.TimePeriod{
			End:   &budgetTimePeriodEnd,
			Start: &budgetTimePeriodStart,
		},
		TimeUnit:    aws.String(budgetTimeUnit),
		CostFilters: budgetCostFilters,
	}
	return budget, nil
}

func decodeBudgetsBudgetID(id string) (string, string, error) {
	parts := strings.Split(id, ":")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("Unexpected format of ID (%q), expected AccountID:BudgetName", id)
	}
	return parts[0], parts[1], nil
}

func expandBudgetsCostTypesUnmarshal(budgetCostTypes []interface{}) *budgets.CostTypes {
	costTypes := &budgets.CostTypes{
		IncludeCredit:            aws.Bool(true),
		IncludeDiscount:          aws.Bool(true),
		IncludeOtherSubscription: aws.Bool(true),
		IncludeRecurring:         aws.Bool(true),
		IncludeRefund:            aws.Bool(true),
		IncludeSubscription:      aws.Bool(true),
		IncludeSupport:           aws.Bool(true),
		IncludeTax:               aws.Bool(true),
		IncludeUpfront:           aws.Bool(true),
		UseAmortized:             aws.Bool(false),
		UseBlended:               aws.Bool(false),
	}
	if len(budgetCostTypes) == 1 {
		costTypesMap := budgetCostTypes[0].(map[string]interface{})
		for k, v := range map[string]*bool{
			"include_credit":             costTypes.IncludeCredit,
			"include_discount":           costTypes.IncludeDiscount,
			"include_other_subscription": costTypes.IncludeOtherSubscription,
			"include_recurring":          costTypes.IncludeRecurring,
			"include_refund":             costTypes.IncludeRefund,
			"include_subscription":       costTypes.IncludeSubscription,
			"include_support":            costTypes.IncludeSupport,
			"include_tax":                costTypes.IncludeTax,
			"include_upfront":            costTypes.IncludeUpfront,
			"use_amortized":              costTypes.UseAmortized,
			"use_blended":                costTypes.UseBlended,
		} {
			if val, ok := costTypesMap[k]; ok {
				*v = val.(bool)
			}
		}
	}

	return costTypes
}
