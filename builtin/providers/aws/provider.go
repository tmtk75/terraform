package aws

import (
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

// Provider returns a terraform.ResourceProvider.
func Provider() terraform.ResourceProvider {
	// TODO: Move the validation to this, requires conditional schemas
	// TODO: Move the configuration to this, requires validation

	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"access_key": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{
					"AWS_ACCESS_KEY",
					"AWS_ACCESS_KEY_ID",
				}, nil),
				Description: descriptions["access_key"],
			},

			"secret_key": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{
					"AWS_SECRET_KEY",
					"AWS_SECRET_ACCESS_KEY",
				}, nil),
				Description: descriptions["secret_key"],
			},

			"token": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{
					"AWS_SESSION_TOKEN",
				}, ""),
				Description: descriptions["token"],
			},

			"region": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{
					"AWS_REGION",
					"AWS_DEFAULT_REGION",
				}, nil),
				Description:  descriptions["region"],
				InputDefault: "us-east-1",
			},

			"allowed_account_ids": &schema.Schema{
				Type:          schema.TypeSet,
				Elem:          &schema.Schema{Type: schema.TypeString},
				Optional:      true,
				ConflictsWith: []string{"forbidden_account_ids"},
				Set: func(v interface{}) int {
					return hashcode.String(v.(string))
				},
			},

			"forbidden_account_ids": &schema.Schema{
				Type:          schema.TypeSet,
				Elem:          &schema.Schema{Type: schema.TypeString},
				Optional:      true,
				ConflictsWith: []string{"allowed_account_ids"},
				Set: func(v interface{}) int {
					return hashcode.String(v.(string))
				},
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"aws_autoscaling_group":            resourceAwsAutoscalingGroup(),
			"aws_db_instance":                  resourceAwsDbInstance(),
			"aws_db_parameter_group":           resourceAwsDbParameterGroup(),
			"aws_db_security_group":            resourceAwsDbSecurityGroup(),
			"aws_db_subnet_group":              resourceAwsDbSubnetGroup(),
			"aws_elasticache":                  resourceAwsElasticache(),
			"aws_elasticache_subnet_group":     resourceAwsElasticacheSubnetGroup(),
			"aws_elasticache_security_group":   resourceAwsElasticacheSecurityGroup(),
			"aws_eip":                          resourceAwsEip(),
			"aws_elb":                          resourceAwsElb(),
			"aws_instance":                     resourceAwsInstance(),
			"aws_internet_gateway":             resourceAwsInternetGateway(),
			"aws_key_pair":                     resourceAwsKeyPair(),
			"aws_launch_configuration":         resourceAwsLaunchConfiguration(),
			"aws_main_route_table_association": resourceAwsMainRouteTableAssociation(),
			"aws_network_acl":                  resourceAwsNetworkAcl(),
			"aws_network_interface":            resourceAwsNetworkInterface(),
			"aws_route53_record":               resourceAwsRoute53Record(),
			"aws_route53_zone":                 resourceAwsRoute53Zone(),
			"aws_route_table":                  resourceAwsRouteTable(),
			"aws_route_table_association":      resourceAwsRouteTableAssociation(),
			"aws_s3_bucket":                    resourceAwsS3Bucket(),
			"aws_security_group":               resourceAwsSecurityGroup(),
			"aws_subnet":                       resourceAwsSubnet(),
			"aws_vpc":                          resourceAwsVpc(),
			"aws_vpc_peering_connection":       resourceAwsVpcPeeringConnection(),
			"aws_vpn_gateway":                  resourceAwsVpnGateway(),
		},

		ConfigureFunc: providerConfigure,
	}
}

var descriptions map[string]string

func init() {
	descriptions = map[string]string{
		"region": "The region where AWS operations will take place. Examples\n" +
			"are us-east-1, us-west-2, etc.",

		"access_key": "The access key for API operations. You can retrieve this\n" +
			"from the 'Security & Credentials' section of the AWS console.",

		"secret_key": "The secret key for API operations. You can retrieve this\n" +
			"from the 'Security & Credentials' section of the AWS console.",

		"token": "session token. A session token is only required if you are\n" +
			"using temporary security credentials.",
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	config := Config{
		AccessKey: d.Get("access_key").(string),
		SecretKey: d.Get("secret_key").(string),
		Token:     d.Get("token").(string),
		Region:    d.Get("region").(string),
	}

	if v, ok := d.GetOk("allowed_account_ids"); ok {
		config.AllowedAccountIds = v.(*schema.Set).List()
	}

	if v, ok := d.GetOk("forbidden_account_ids"); ok {
		config.ForbiddenAccountIds = v.(*schema.Set).List()
	}

	return config.Client()
}
