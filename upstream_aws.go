package main

import (
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
)

type AutoScalingGroupUpstreamSource struct {
	AutoScalingGroupId string
	Region string
}

// UpdatePeers uses the AWS SDK to update the list of peers available in an AutoScalingGroup
func (as AutoScalingGroupUpstreamSource) UpdatePeers() ([]string, error) {
	sess := session.Must(session.NewSession())
	asg := autoscaling.New(sess, &aws.Config{Region: aws.String(as.Region)})
	ec := ec2.New(sess, &aws.Config{Region: aws.String(as.Region)})

	group, err := asg.DescribeAutoScalingGroups(&autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: []*string{aws.String(as.AutoScalingGroupId)},
	})

	if err != nil {
		return nil, err
	}

	if len(group.AutoScalingGroups) != 1  {
		return nil, errors.New(fmt.Sprintf("No AutoScalingGroup found '%s'", as.AutoScalingGroupId))
	}

	ids := make([]*string, len(group.AutoScalingGroups[0].Instances))
	for i, inst := range group.AutoScalingGroups[0].Instances {
		ids[i] = inst.InstanceId
	}

	instances, err := ec.DescribeInstances(&ec2.DescribeInstancesInput{
		InstanceIds: ids,
		Filters: []*ec2.Filter{
			{
				Name: aws.String("instance-state-name"),
				Values: []*string{ aws.String("running") },
			},
		},
	})

	if err != nil {
		return nil, err
	}

	peers := make([]string, len(ids))
	j := 0
	for _, reservation := range instances.Reservations {
		for _, inst := range reservation.Instances {
			peers[j] = *inst.PrivateIpAddress
			j++
		}
	}

	return peers, nil
}


func NewAutoScalingGroupUpstreamSource(region string, asgID string) (*AutoScalingGroupUpstreamSource, error) {
	return &AutoScalingGroupUpstreamSource{
		Region: region,
		AutoScalingGroupId: asgID,
	}, nil
}

func MustNewAutoScalingGroupUpstreamSource(region string, asgID string) *AutoScalingGroupUpstreamSource {
	asgus, err := NewAutoScalingGroupUpstreamSource(region, asgID)
	if err != nil {
		panic(err)
	}

	return asgus
}
