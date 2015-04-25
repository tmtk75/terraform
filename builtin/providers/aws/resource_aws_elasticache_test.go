package aws

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/service/elasticache"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSElasticache(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSElasticacheDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSElasticacheConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEcacheSecurityGroupExists("aws_elasticache_security_group.bar"),
					testAccCheckAWSElasticacheExists("aws_elasticache.bar"),
				),
			},
			resource.TestStep{
				Config: testAccAWSElasticacheInVPCConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSEcacheSubnetGroupExists("aws_elasticache_subnet_group.bar"),
					testAccCheckAWSElasticacheExists("aws_elasticache.bar"),
				),
			},
		},
	})
}

func testAccCheckAWSElasticacheDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).elasticacheconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_elasticache" {
			continue
		}
		res, err := conn.DescribeCacheClusters(&elasticache.DescribeCacheClustersInput{
			CacheClusterID: aws.String(rs.Primary.ID),
		})
		if err != nil {
			return err
		}
		if len(res.CacheClusters) > 0 {
			return fmt.Errorf("still exist.")
		}
	}
	return nil
}

func testAccCheckAWSElasticacheExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No cache cluster ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).elasticacheconn
		_, err := conn.DescribeCacheClusters(&elasticache.DescribeCacheClustersInput{
			CacheClusterID: aws.String(rs.Primary.ID),
		})
		if err != nil {
			return fmt.Errorf("Elasticache error: %v", err)
		}
		return nil
	}
}

func genRandInt() int {
	return rand.New(rand.NewSource(time.Now().UnixNano())).Int() % 1000
}

var testAccAWSElasticacheConfig = fmt.Sprintf(`
resource "aws_security_group" "bar" {
    name = "tf-test-security-group-%03d"
    description = "tf-test-security-group-descr"
    ingress {
        from_port = -1
        to_port = -1
        protocol = "icmp"
        cidr_blocks = ["0.0.0.0/0"]
    }
}

resource "aws_elasticache_security_group" "bar" {
    name = "tf-test-security-group-%03d"
    description = "tf-test-security-group-descr"
    security_group_names = ["${aws_security_group.bar.name}"]
}

resource "aws_elasticache" "bar" {
    cluster_id = "tf-test-%03d"
    engine = "memcached"
    node_type = "cache.m1.small"
    num_cache_nodes = 1
    parameter_group_name = "default.memcached1.4"
    security_group_names = ["${aws_elasticache_security_group.bar.name}"]
}
`, genRandInt(), genRandInt(), genRandInt())

var testAccAWSElasticacheInVPCConfig = fmt.Sprintf(`
resource "aws_vpc" "foo" {
    cidr_block = "192.168.0.0/16"
    tags {
            Name = "tf-test"
    }
}

resource "aws_subnet" "foo" {
    vpc_id = "${aws_vpc.foo.id}"
    cidr_block = "192.168.0.0/20"
    availability_zone = "us-west-2a"
    tags {
            Name = "tf-test"
    }
}

resource "aws_elasticache_subnet_group" "bar" {
    name = "tf-test-cache-subnet-%03d"
    description = "tf-test-cache-subnet-group-descr"
    subnet_ids = ["${aws_subnet.foo.id}"]
}

resource "aws_security_group" "bar" {
    name = "tf-test-security-group-%03d"
    description = "tf-test-security-group-descr"
    vpc_id = "${aws_vpc.foo.id}"
    ingress {
        from_port = -1
        to_port = -1
        protocol = "icmp"
        cidr_blocks = ["0.0.0.0/0"]
    }
}

resource "aws_elasticache" "bar" {
    cluster_id = "tf-test-%03d"
    node_type = "cache.m1.small"
    num_cache_nodes = 1
    engine = "redis"
    engine_version = "2.8.19"
    port = 6379
    subnet_group_name = "${aws_elasticache_subnet_group.bar.name}"
    security_group_ids = ["${aws_security_group.bar.id}"]
    parameter_group_name = "default.redis2.8"
}
`, genRandInt(), genRandInt(), genRandInt())
