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

func resourceAwsElasticache() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsElasticacheCreate,
		Read:   resourceAwsElasticacheRead,
		Update: resourceAwsElasticacheUpdate,
		Delete: resourceAwsElasticacheDelete,

		Schema: map[string]*schema.Schema{
			"cluster_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"engine": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"node_type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"num_cache_nodes": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},
			"parameter_group_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"port": &schema.Schema{
				Type:     schema.TypeInt,
				Default:  11211,
				Optional: true,
				ForceNew: true,
			},
			"engine_version": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"subnet_group_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"security_group_names": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set: func(v interface{}) int {
					return hashcode.String(v.(string))
				},
			},
			"security_group_ids": &schema.Schema{
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

func resourceAwsElasticacheCreate(d *schema.ResourceData, meta interface{}) error {
	elasticacheconn := meta.(*AWSClient).elasticacheconn

	clusterId := d.Get("cluster_id").(string)
	nodeType := d.Get("node_type").(string)           // e.g) cache.m1.small
	numNodes := d.Get("num_cache_nodes").(int)        // 2
	engine := d.Get("engine").(string)                // memcached
	engineVersion := d.Get("engine_version").(string) // 1.4.14
	port := d.Get("port").(int)                       // 11211
	subnetGroupName := d.Get("subnet_group_name").(string)
	securityNameSet := d.Get("security_group_names").(*schema.Set)
	securityIdSet := d.Get("security_group_ids").(*schema.Set)
	paramGroupName := d.Get("parameter_group_name").(string) // default.memcached1.4

	securityNames := make([]string, securityNameSet.Len())
	for i, name := range securityNameSet.List() {
		securityNames[i] = name.(string)
	}
	securityIds := make([]string, securityIdSet.Len())
	for i, id := range securityIdSet.List() {
		securityIds[i] = id.(string)
	}

	req := &elasticcache.CreateCacheClusterMessage{
		CacheClusterID:          aws.String(clusterId),
		CacheNodeType:           aws.String(nodeType),
		NumCacheNodes:           aws.Integer(numNodes),
		Engine:                  aws.String(engine),
		EngineVersion:           aws.String(engineVersion),
		Port:                    aws.Integer(port),
		CacheSubnetGroupName:    aws.String(subnetGroupName),
		CacheSecurityGroupNames: securityNames,
		SecurityGroupIDs:        securityIds,
		CacheParameterGroupName: aws.String(paramGroupName),
	}

	_, err := elasticacheconn.CreateCacheCluster(req)
	if err != nil {
		return fmt.Errorf("Error creating Elasticache: %s", err)
	}

	d.SetId(clusterId)

	return nil
}

func resourceAwsElasticacheRead(d *schema.ResourceData, meta interface{}) error {
	elasticacheconn := meta.(*AWSClient).elasticacheconn
	req := &elasticcache.DescribeCacheClustersMessage{
		CacheClusterID: aws.String(d.Id()),
	}

	res, err := elasticacheconn.DescribeCacheClusters(req)
	if err != nil {
		return err
	}

	if len(res.CacheClusters) == 1 {
		c := res.CacheClusters[0]
		d.Set("cluster_id", c.CacheClusterID)
		d.Set("node_type", c.CacheNodeType)
		d.Set("num_cache_nodes", c.NumCacheNodes)
		d.Set("engine", c.Engine)
		d.Set("engine_version", c.EngineVersion)
		if c.ConfigurationEndpoint != nil {
			d.Set("port", c.ConfigurationEndpoint.Port)
		}
		d.Set("subnet_group_name", c.CacheSubnetGroupName)
		d.Set("security_group_names", c.CacheSecurityGroups)
		d.Set("security_group_ids", c.SecurityGroups)
		d.Set("parameter_group_name", c.CacheParameterGroup)
	}

	return nil
}

func resourceAwsElasticacheUpdate(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceAwsElasticacheDelete(d *schema.ResourceData, meta interface{}) error {
	elasticacheconn := meta.(*AWSClient).elasticacheconn

	// AWS CLI complains like below if we try to delete while creating.
	//     Can only delete cache clusters with state in:
	//     available, failed, incompatible-parameters, incompatible-network, restore-failed
	// So we cannot delete clusters in `creating` state.
	// We need to wait for states which we can delete.
	pending := []string{"creating"}
	stateConf := &resource.StateChangeConf{
		Pending:    pending,
		Target:     "available",
		Refresh:    CacheClusterStateRefreshFunc(elasticacheconn, d.Id(), "available", pending),
		Timeout:    10 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	log.Printf("[DEBUG] Waiting for state to become available: %v", d.Id())
	_, sterr := stateConf.WaitForState()
	if sterr != nil {
		return fmt.Errorf("Error waiting for elasticache (%s) to be deletable: %s", d.Id(), sterr)
	}

	req := &elasticcache.DeleteCacheClusterMessage{
		CacheClusterID: aws.String(d.Id()),
	}
	_, err := elasticacheconn.DeleteCacheCluster(req)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Waiting for deletion: %v", d.Id())
	stateConf = &resource.StateChangeConf{
		Pending:    []string{"creating", "available", "deleting", "incompatible-parameters", "incompatible-network", "restore-failed"},
		Target:     "",
		Refresh:    CacheClusterStateRefreshFunc(elasticacheconn, d.Id(), "", []string{}),
		Timeout:    10 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, sterr = stateConf.WaitForState()
	if sterr != nil {
		return fmt.Errorf("Error waiting for elasticache (%s) to delete: %s", d.Id(), sterr)
	}

	d.SetId("")

	return nil
}

func CacheClusterStateRefreshFunc(conn *elasticcache.ElasticCache, clusterID, givenState string, pending []string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		resp, err := conn.DescribeCacheClusters(&elasticcache.DescribeCacheClustersMessage{
			CacheClusterID: aws.String(clusterID),
		})
		if err != nil {
			apierr := err.(aws.APIError)
			log.Printf("[DEBUG] message: %v, code: %v", apierr.Message, apierr.Code)
			if apierr.Message == fmt.Sprintf("CacheCluster not found: %v", clusterID) {
				log.Printf("[DEBUG] Detect deletion")
				return nil, "", nil
			}

			log.Printf("[ERROT] CacheClusterStateRefreshFunc: %s", err)
			return nil, "", err
		}

		c := resp.CacheClusters[0]
		log.Printf("[DEBUG] status: %v", *c.CacheClusterStatus)

		// return the current state if it's in the pending array
		for _, p := range pending {
			s := *c.CacheClusterStatus
			if p == s {
				log.Printf("[DEBUG] Return with status: %v", *c.CacheClusterStatus)
				return c, p, nil
			}
		}

		// return given state if it's not in pending
		if givenState != "" {
			return c, givenState, nil
		}
		log.Printf("[DEBUG] current status: %v", *c.CacheClusterStatus)
		return c, *c.CacheClusterStatus, nil
	}
}
