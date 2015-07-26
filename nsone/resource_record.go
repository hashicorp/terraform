package nsone

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/bobtfish/go-nsone-api"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
	"regexp"
	"strings"
)

func recordResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"zone": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"domain": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, es []error) {
					value := v.(string)
					if !regexp.MustCompile(`^(A|AAAA|ALIAS|AFSDB|CNAME|DNAME|HINFO|MX|NAPTR|NS|PTR|RP|SPF|SRV|TXT)$`).MatchString(value) {
						es = append(es, fmt.Errorf(
							"only A, AAAA, ALIAS, AFSDB, CNAME, DNAME, HINFO, MX, NAPTR, NS, PTR, RP, SPF, SRV, TXT allowed in %q", k))
					}
					return
				},
			},
			"meta": metaSchema(),
			"link": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				Computed:      true,
				ConflictsWith: []string{"meta", "answers"},
			},
			"answers": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"answer": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							Computed: true,
						},
						"meta": &schema.Schema{
							Type:     schema.TypeSet,
							Optional: true,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"field": &schema.Schema{
										Type:     schema.TypeString,
										Required: true,
										Computed: true,
									},
									"feed": &schema.Schema{
										Type:     schema.TypeString,
										Required: true,
										Computed: true,
									},
								},
							},
							Set: metaToHash,
						},
					},
				},
				Set: answersToHash,
			},
		},
		Create: RecordCreate,
		Read:   RecordRead,
		Update: RecordUpdate,
		Delete: RecordDelete,
	}
}

func answersToHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["answer"].(string)))

	return hashcode.String(buf.String())
}

func metaToHash(v interface{}) int {
	var buf bytes.Buffer
	s := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", s["field"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", s["feed"].(string)))

	return hashcode.String(buf.String())
}

func recordToResourceData(d *schema.ResourceData, r *nsone.Record) {
	d.SetId(r.Id)
	d.Set("domain", r.Domain)
	d.Set("zone", r.Zone)
	d.Set("type", r.Type)
	if r.Link != "" {
		d.Set("link", r.Link)
	}
}

func setToMapByKey(s *schema.Set, key string) map[string]interface{} {
	result := make(map[string]interface{})
	for _, rawData := range s.List() {
		data := rawData.(map[string]interface{})
		result[data[key].(string)] = data
	}

	return result
}

func RecordCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*nsone.APIClient)
	r := nsone.NewRecord(d.Get("zone").(string), d.Get("domain").(string), d.Get("type").(string))
	if answers := d.Get("answers").(*schema.Set); answers.Len() > 0 {
		al := make([]nsone.Answer, 0)
		for _, answer_raw := range answers.List() {
			answer := answer_raw.(map[string]interface{})
			a := nsone.NewAnswer()
			v := answer["answer"].(string)
			if d.Get("type") != "TXT" {
				a.Answer = strings.Split(v, " ")
			} else {
				a.Answer = []string{v}
			}
			if metas := answer["meta"].(*schema.Set); metas.Len() > 0 {
				for _, meta_raw := range metas.List() {
					meta := meta_raw.(map[string]interface{})
					key := meta["field"].(string)
					value := meta["feed"].(string)
					a.Meta[key] = nsone.NewMetaFeed(value)
				}
			}
			al = append(al, a)
		}
		r.Answers = al
		if _, ok := d.GetOk("link"); ok {
			return errors.New("Cannot have both link and answers in a record")
		}
	}
	if v, ok := d.GetOk("link"); ok {
		r.LinkTo(v.(string))
	}
	err := client.CreateRecord(r)
	//    zone := d.Get("zone").(string)
	//    hostmaster := d.Get("hostmaster").(string)
	if err != nil {
		return err
	}
	recordToResourceData(d, r)
	return nil
}

func RecordRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*nsone.APIClient)
	r, err := client.GetRecord(d.Get("zone").(string), d.Get("domain").(string), d.Get("type").(string))
	//    zone := d.Get("zone").(string)
	//    hostmaster := d.Get("hostmaster").(string)
	if err != nil {
		return err
	}
	recordToResourceData(d, r)
	return nil
}

func RecordDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*nsone.APIClient)
	err := client.DeleteRecord(d.Get("zone").(string), d.Get("domain").(string), d.Get("type").(string))
	d.SetId("")
	return err
}

func RecordUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*nsone.APIClient)
	r := nsone.NewRecord(d.Get("zone").(string), d.Get("domain").(string), d.Get("type").(string))
	r.Id = d.Id()
	err := client.UpdateRecord(r)
	if err != nil {
		return err
	}
	recordToResourceData(d, r)
	return nil
}
