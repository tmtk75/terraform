package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/aws-sdk-go/aws"
	"github.com/hashicorp/aws-sdk-go/gen/elasticache"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsElasticacheSubnetGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsElasticacheSubnetGroupCreate,
		Read:   resourceAwsElasticacheSubnetGroupRead,
		Delete: resourceAwsElasticacheSubnetGroupDelete,

		Schema: map[string]*schema.Schema{
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"subnet_ids": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set: func(v interface{}) int {
					return hashcode.String(v.(string))
				},
			},
		},
	}
}

func resourceAwsElasticacheSubnetGroupCreate(d *schema.ResourceData, meta interface{}) error {
	elasticacheconn := meta.(*AWSClient).elasticacheconn

	// Get the group properties
	name := d.Get("name").(string)
	desc := d.Get("description").(string)
	subnetIdsSet := d.Get("subnet_ids").(*schema.Set)

	log.Printf("[DEBUG] Cache subnet group create: name: %s, description: %s", name, desc)

	subnetIds := make([]string, subnetIdsSet.Len())
	for i, subnetId := range subnetIdsSet.List() {
		subnetIds[i] = subnetId.(string)
	}

	req := &elasticcache.CreateCacheSubnetGroupMessage{
		CacheSubnetGroupDescription: aws.String(desc),
		CacheSubnetGroupName:        aws.String(name),
		SubnetIDs:                   subnetIds,
	}

	_, err := elasticacheconn.CreateCacheSubnetGroup(req)
	if err != nil {
		return fmt.Errorf("Error creating CacheSubnetGroup: %s", err)
	}

	// Assign the group name as the resource ID
	d.SetId(name)

	return nil
}

func resourceAwsElasticacheSubnetGroupRead(d *schema.ResourceData, meta interface{}) error {
	elasticacheconn := meta.(*AWSClient).elasticacheconn
	req := &elasticcache.DescribeCacheSubnetGroupsMessage{
		CacheSubnetGroupName: aws.String(d.Get("name").(string)),
	}

	res, err := elasticacheconn.DescribeCacheSubnetGroups(req)
	if err != nil {
		return err
	}
	if len(res.CacheSubnetGroups) == 0 {
		return fmt.Errorf("Error missing %v", d.Get("name"))
	}

	var group *elasticcache.CacheSubnetGroup
	for _, g := range res.CacheSubnetGroups {
		log.Printf("[DEBUG] %v %v", g.CacheSubnetGroupName, d.Id())
		if *g.CacheSubnetGroupName == d.Id() {
			group = &g
		}
	}
	if group == nil {
		return fmt.Errorf("Error retrieving cache subnet group: %v", res)
	}

	ids := make([]string, len(group.Subnets))
	for i, s := range group.Subnets {
		ids[i] = *s.SubnetIdentifier
	}

	d.Set("name", group.CacheSubnetGroupName)
	d.Set("description", group.CacheSubnetGroupDescription)
	d.Set("subnet_ids", ids)

	return nil
}

func resourceAwsElasticacheSubnetGroupDelete(d *schema.ResourceData, meta interface{}) error {
	elasticacheconn := meta.(*AWSClient).elasticacheconn

	log.Printf("[DEBUG] Cache subnet group delete: %s", d.Id())

	return resource.Retry(5*time.Minute, func() error {
		err := elasticacheconn.DeleteCacheSubnetGroup(&elasticcache.DeleteCacheSubnetGroupMessage{
			CacheSubnetGroupName: aws.String(d.Id()),
		})
		if err != nil {
			apierr, ok := err.(aws.APIError)
			if !ok {
				return err
			}
			log.Printf("[DEBUG] APIError.Code: %v", apierr.Code)
			switch apierr.Code {
			case "DependencyViolation":
				// If it is a dependency violation, we want to retry
				return err
			default:
				return resource.RetryError{Err: err}
			}
		}
		return nil
	})
}
