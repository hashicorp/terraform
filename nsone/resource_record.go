package nsone

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/bobtfish/go-nsone-api"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
	"log"
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
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},
			"answers": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"answer": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
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
										Computed: true,
										Optional: true,
									},
									"feed": &schema.Schema{
										Type:     schema.TypeString,
										Computed: true,
										Optional: true,
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

func recordToResourceData(d *schema.ResourceData, r *nsone.Record) error {
	d.SetId(r.Id)
	d.Set("domain", r.Domain)
	d.Set("zone", r.Zone)
	d.Set("type", r.Type)
	if r.Link != "" {
		d.Set("link", r.Link)
	}
	log.Println("BOOOO")
	log.Println(r)
	if len(r.Answers) > 0 {
		answers := make([]map[string]interface{}, 0, len(r.Answers))
		log.Println("Got answers")
		for _, answer := range r.Answers {
			log.Println("GOT ANSWER")
			log.Println(answer)
			answers = append(answers, answerToMap(answer))
		}
		log.Println(fmt.Sprintf("[DEBUG] SETTING ANSWERS: %v", answers))
		err := d.Set("answers", &answers)
		if err != nil {
			return fmt.Errorf("[DEBUG] Error setting answers for: %s, error: %#v", r.Domain, err)
		}
	}
	return nil
}

func answerToMap(a nsone.Answer) map[string]interface{} {
	m := make(map[string]interface{})
	m["meta"] = make([]map[string]interface{}, 0)
	m["answer"] = strings.Join(a.Answer, " ")
	if a.Meta != nil {
		metas := make([]map[string]interface{}, len(a.Meta))
		for k, v := range a.Meta {
			meta := make(map[string]interface{})
			meta["field"] = k
			meta["feed"] = v.Feed
			metas = append(metas, meta)
		}
		m["meta"] = metas
	}
	log.Println(fmt.Sprintf("answerToMap %v", m))
	return m
}

func resourceDataToRecord(r *nsone.Record, d *schema.ResourceData) error {
	r.Id = d.Id()
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
	return nil
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
	err := resourceDataToRecord(r, d)
	if err != nil {
		return err
	}
	err = client.CreateRecord(r)
	if err != nil {
		return err
	}
	return recordToResourceData(d, r)
}

func RecordRead(d *schema.ResourceData, meta interface{}) error {
	//z := d.Get("zone").(string)
	//do := d.Get("domain").(string)
	//t := d.Get("type").(string)
	//before := nsone.NewRecord(z, do, t)
	//resourceDataToRecord(before, d)
	//log.Println(fmt.Sprintf("FOOOO READ RECORD %+v", before))
	client := meta.(*nsone.APIClient)
	r, err := client.GetRecord(d.Get("zone").(string), d.Get("domain").(string), d.Get("type").(string))
	if err != nil {
		return err
	}
	recordToResourceData(d, r)
	//after := nsone.NewRecord(z, do, t)
	//resourceDataToRecord(after, d)
	//log.Println(fmt.Sprintf("FINISH READ RECORD %+v", after))
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
	err := resourceDataToRecord(r, d)
	if err != nil {
		return err
	}
	err = client.UpdateRecord(r)
	if err != nil {
		return err
	}
	recordToResourceData(d, r)
	return nil
}
