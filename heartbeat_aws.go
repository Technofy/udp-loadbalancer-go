package main

import (
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"

	"github.com/technofy/udp-loadbalancer-go/config"
)

type PacemakerAws struct {
	cw *cloudwatch.CloudWatch
	Namespace string
	Metric string

	Dimension *cloudwatch.Dimension
}

func (p *PacemakerAws) Heartbeat() {
	aliveValue := float64(1)

	_, err := p.cw.PutMetricData(&cloudwatch.PutMetricDataInput{
		Namespace: aws.String(p.Namespace),
		MetricData: []*cloudwatch.MetricDatum{
			{
				Dimensions: []*cloudwatch.Dimension{
					p.Dimension,
				},
				MetricName: aws.String(p.Metric),
				Unit: aws.String("Count"),
				Value: &aliveValue,
			},
		},
	})

	if err != nil {
		panic(err)
	}
}

// AutoHeartbeat is a helper function that will send a heartbeat to AWS CloudWatch at a regular rate defined by
// the parameter seconds.
func (pm *PacemakerAws) AutoHeartbeatAws(seconds int) {
	ticker := time.NewTicker(time.Second * time.Duration(seconds))

	pm.Heartbeat()
	for range ticker.C {
		pm.Heartbeat()
	}
}

func NewPacemakerAws(region string, namespace string, metric string) (*PacemakerAws, error) {
	sess, err := session.NewSession()
	if err != nil {
		return nil, err
	}

	awscfg := &aws.Config{Region: aws.String(region)}
	meta := ec2metadata.New(sess, awscfg)

	instid, err := meta.GetMetadata("instance-id")

	if err != nil {
		return nil, err
	}

	p := &PacemakerAws{
		cw: cloudwatch.New(sess, awscfg),
		Namespace: namespace,
		Metric: metric,
		Dimension: &cloudwatch.Dimension{
			Name: aws.String("InstanceId"),
			Value: aws.String(instid),
		},
	}

	return p, nil
}

func NewPacemakerAwsFromConfig(pm *config.Pacemaker) (*PacemakerAws, error) {
	sess, err := session.NewSession()

	if err != nil {
		return nil, err
	}

	awscfg := &aws.Config{Region: aws.String(pm.Region)}
	meta := ec2metadata.New(sess, awscfg)

	instid := pm.DimensionValue
	if len(instid) == 0 {
		instid, err = meta.GetMetadata("instance-id")

		if err != nil {
			return nil, err
		}
	}

	p := &PacemakerAws{
		cw: cloudwatch.New(sess, awscfg),
		Namespace: pm.Namespace,
		Metric: pm.Metric,
		Dimension: &cloudwatch.Dimension{
			Name: aws.String("InstanceId"),
			Value: aws.String(instid),
		},
	}

	return p, nil
}