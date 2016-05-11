package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/jeremyd/easyssh"
	"time"
	"bufio"
	"os"
	"flag"
	"io/ioutil"
	"github.com/spf13/viper"
	"strings"
)

func req_spot(client *ec2.EC2, ami_id string) (*string) {
	price := viper.GetString("build.spot_price")
	sec_group_ids := make([]*string, 1)
	sec0 := viper.GetString("build.security_group_id")
	sec_group_ids[0] = &sec0

	keyname := viper.GetString("build.ssh_key_name")
	//instancetype := ec2.InstanceTypeM3Large
	instancetype := string(viper.GetString("build.instance_type"))
	lspec := ec2.RequestSpotLaunchSpecification{
		KeyName: &keyname,
		//UserData: ""54.184.34.9,
		ImageId: &ami_id,
		InstanceType: aws.String(instancetype),
		SecurityGroupIds: sec_group_ids }
		
	if viper.IsSet("build.subnet_id") {
		lspec.SubnetId = aws.String(viper.GetString("build.subnet_id"))
	}

	params := ec2.RequestSpotInstancesInput{
		SpotPrice: &price,
		LaunchSpecification: &lspec }

	req, resp := client.RequestSpotInstancesRequest(&params)
	err := req.Send()
	if err == nil { // resp is now filled
		fmt.Println(resp)
	} else {
		fmt.Println(err)
		fmt.Println(resp)
	}
	return resp.SpotInstanceRequests[0].SpotInstanceRequestId
}

func cancel_spot_req(client *ec2.EC2, spot_req_id *string) {
	params := &ec2.CancelSpotInstanceRequestsInput{
		SpotInstanceRequestIds: []*string{ 
			spot_req_id,
		},
	}
	_, err := client.CancelSpotInstanceRequests(params)

	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
	}
}

func wait_for_spot_req_active(client *ec2.EC2, spot_req_id *string) (instance_id *string) {
	params := &ec2.DescribeSpotInstanceRequestsInput{
		SpotInstanceRequestIds: []*string{ spot_req_id } }

	for {
		resp, err := client.DescribeSpotInstanceRequests(params)
		// If there is a problem with the state of the Spot instance then delete state and quit.
		if err != nil {
			fmt.Println(err.Error())
			fmt.Println("ERROR: Could not determine state of spot instance.  Aborting and clearing cache spot instance id. (Retry?)")
			os.Remove(".buildami.state")
			panic("exiting")
		}
		if *resp.SpotInstanceRequests[0].State == ec2.SpotInstanceStateClosed {
			fmt.Println("ERROR: spot instance state was closed.  Aborting and clearing cache spot instance id. (Retry?)")
			os.Remove(".buildami.state")
			panic("exiting")
		}
		if *resp.SpotInstanceRequests[0].Status.Code == "price-too-low" {
			fmt.Println("ERROR: spot instance price was set too low.")
			fmt.Println(*resp.SpotInstanceRequests[0].Status.Message)
			cancel_spot_req(client, spot_req_id)
			os.Remove(".buildami.state")
			panic("exiting")
		}		
		if *resp.SpotInstanceRequests[0].State == ec2.SpotInstanceStateActive {
			return resp.SpotInstanceRequests[0].InstanceId
		}
		time.Sleep(5 * time.Second)
	}
}

func get_instance_ip(client *ec2.EC2, id *string) (ip *string) {
	params := &ec2.DescribeInstancesInput{
		InstanceIds: []*string{ id } }

	resp, err := client.DescribeInstances(params)

	if err != nil {
		fmt.Println(err.Error())
	}
	//fmt.Println(resp)
	return resp.Reservations[0].Instances[0].PublicIpAddress
}

func write_spot_state(spot_req_id *string) {
		d1 := []byte(*spot_req_id)
    err := ioutil.WriteFile(".hello.state", d1, 0644)
		if err != nil {
			fmt.Println("WARNING: error occured writing to .hello.state file")
		}
}

func create_and_attach_volume(client *ec2.EC2, instance_id *string) (volume_id *string) {
	// DESCRIBE INSTANCE TO GET METADATA
	iparams := &ec2.DescribeInstancesInput{
		InstanceIds: []*string{ instance_id } }
	iresp, ierr := client.DescribeInstances(iparams)
	if ierr != nil {
		fmt.Println(ierr.Error())
		return
	}
	//fmt.Println(iresp)
	availability_zone := iresp.Reservations[0].Instances[0].Placement.AvailabilityZone
	
	image_size := int64(viper.GetInt("build.image_size"))

	// CREATE VOLUME
	vparams := &ec2.CreateVolumeInput{
		AvailabilityZone: availability_zone, // Required
		Size:             aws.Int64(image_size),
		VolumeType:       aws.String("gp2"),
	}
	vresp, verr := client.CreateVolume(vparams)
	if verr != nil {
		fmt.Println(verr.Error())
		return
	}
	//fmt.Println(vresp)
	save_volume_id := vresp.VolumeId

	// ATTACH VOLUME TO INSTANCE
	keep_waiting_attach := true
	for keep_waiting_attach {
		avparams := &ec2.AttachVolumeInput{
			Device:     aws.String("/dev/sdx"),
			InstanceId: instance_id,
			VolumeId:   save_volume_id }
		_, averr := client.AttachVolume(avparams)
		if averr != nil {
			fmt.Println(averr.Error())
			time.Sleep(time.Second * 10)
		} else {
			keep_waiting_attach = false
		}
		//fmt.Println(avresp)
	}
	return save_volume_id
}

func ssh_cmd(instance_ip string, command string) (success bool) {
	ssh := &easyssh.MakeConfig{
		User:   "root",
		Server: instance_ip,
		// Optional key or Password without either we try to contact your agent SOCKET
		Key:  viper.GetString("build.ssh_key_path"),
		Port: "22",
	}
	// Call Run method with command you want to run on remote server.
	_, err := ssh.Run(command)

	// Handle errors
	if err != nil {
		fmt.Println("Can't run remote command: " + err.Error())
		//fmt.Println(response)
		return false
	} 
	//fmt.Println(response)
	return true
}

func ssh_cp(instance_ip string, file string) {
	ssh := &easyssh.MakeConfig{
		User:   "root",
		Server: instance_ip,
		// Optional key or Password without either we try to contact your agent SOCKET
		Key:  viper.GetString("build.ssh_key_path"),
		Port: "22",
	}
	// Call Scp method with file you want to upload to remote server.
	err := ssh.Scp(file)

	// Handle errors
	if err != nil {
		panic("Can't run remote command: " + err.Error())
	} else {
		fmt.Println("success uploading " + file)
	}
}

func detach_volume_from_instance(client *ec2.EC2, volume_id *string, instance_id *string) (success bool) {
	params := &ec2.DetachVolumeInput{
		VolumeId:   volume_id, // Required
		InstanceId: instance_id }
	_, err := client.DetachVolume(params)
	if err != nil {
		fmt.Println(err.Error())
		return false
	}
	//fmt.Println(resp)
	return true
}

func wait_for_volume_available(client *ec2.EC2, volume_id *string) {
	keep_waiting := true
	for keep_waiting {
		params := &ec2.DescribeVolumesInput{
			VolumeIds: []*string{
				volume_id,
			},
		}
		resp, err := client.DescribeVolumes(params)
		if err != nil {
			fmt.Println(err.Error())
		}
		//fmt.Println(resp)
		if *resp.Volumes[0].State == ec2.VolumeStateAvailable {
			fmt.Println("volume is available")
			break
		}
		time.Sleep(time.Second * 5)
	}
}

func snapshot_volume(client *ec2.EC2, volume_id *string, snap_title string) (snapshot_id *string) {
	params := &ec2.CreateSnapshotInput{
		VolumeId:    volume_id, // Required
		Description: aws.String(snap_title),
	}
	resp, err := client.CreateSnapshot(params)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	//fmt.Println(resp)
	return resp.SnapshotId
}

func wait_snapshot(client *ec2.EC2, snapshot_id *string) {
	params := &ec2.DescribeSnapshotsInput{
		SnapshotIds: []*string{ snapshot_id }}
	err := client.WaitUntilSnapshotCompleted(params)
	if err != nil {
		fmt.Println(err.Error())
	}
}

func create_image(client *ec2.EC2, snapshot_id *string, title string) (image_id *string) {
	image_size := int64(viper.GetInt("build.image_size"))
	params := &ec2.RegisterImageInput{
		Name:         aws.String(title), // Required
		BlockDeviceMappings: []*ec2.BlockDeviceMapping{
			{ // Required
				DeviceName: aws.String("/dev/sda1"),
				Ebs: &ec2.EbsBlockDevice{
					DeleteOnTermination: aws.Bool(true),
					SnapshotId:          snapshot_id,
					VolumeSize:          aws.Int64(image_size),
					VolumeType:          aws.String("gp2"),
				},
			},
			// More values...
		},
		Architecture:				aws.String(ec2.ArchitectureValuesX8664),
		Description:        aws.String(title),
		RootDeviceName:     aws.String("/dev/sda1"),
		//SriovNetSupport:    aws.String("String"),
		VirtualizationType: aws.String("hvm"),
	}
	resp, err := client.RegisterImage(params)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	//fmt.Println(resp)
	return resp.ImageId
}

func cleanup(client *ec2.EC2, volume_id *string) {
	// DELETE VOLUME
	params := &ec2.DeleteVolumeInput{
		VolumeId: volume_id,
	}
	_, err := client.DeleteVolume(params)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
}

func harvest_software_versions(client *ec2.EC2, instance_ip *string) {
	ssh_cmd(*instance_ip, "pacman -Q linux-ec2 2> /dev/null")
	ssh_cmd(*instance_ip, "pacman -Q systemd 2> /dev/null")
	ssh_cmd(*instance_ip, "pacman -Q kubernetes 2> /dev/null")
	ssh_cmd(*instance_ip, "pacman -Q etcd 2> /dev/null")
	ssh_cmd(*instance_ip, "pacman -Q rkt 2> /dev/null")
	ssh_cmd(*instance_ip, "pacman -Q docker 2> /dev/null")
}

func main() {
	// Viper configuration engine
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	viper.SetEnvPrefix("BB")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	err := viper.ReadInConfig() // Find and read the config file
	if err != nil { // Handle errors reading the config file
	    panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}
	
	// Flags
	var interactive_flag = flag.Bool("interactive", viper.GetBool("build.interactive"), "pause for user interaction")
	var image_prefix = flag.String("image-prefix", viper.GetString("publish.image_prefix"), "the basename of the image to create")
	var payload_script_name = flag.String("payload-script", viper.GetString("build.payload_script"), "name of build script in cwd")
	var use_existing_builder = flag.Bool("use-existing-builder", true, "use the existing build instance")
	flag.Parse()
	
	// Main Logic
	t := time.Now()
	timestamp := t.Format("2006-01-02-150405")
	image_title := *image_prefix + timestamp
	svc := ec2.New(session.New(), &aws.Config{Region: aws.String(viper.GetString("build.region"))})
	// Check for existing builder
	spot_req_id := aws.String("")
	dat, err := ioutil.ReadFile(".buildami.state")
	if *use_existing_builder && err == nil {
		fmt.Println("Using existing builder from " + string(dat))
		spot_req_id = aws.String(string(dat))
	} else {
		fmt.Println("Requesting spot instance...")
		spot_req_id = req_spot(svc, viper.GetString("build.ami_id"))
		write_spot_state(spot_req_id)
	}
	//spot_req_id := aws.String("sir-03hfzd70")
	fmt.Println("Waiting for spot instance to become active...")
	instance_id := wait_for_spot_req_active(svc, spot_req_id)
  	// get it's IP address
	fmt.Println("Getting instance ip address...")
	instance_ip := get_instance_ip(svc, instance_id)
	fmt.Println("Instance IP address: " + *instance_ip)
	// create and attach
	fmt.Println("Creating and attaching new volume...")
	volume_id := create_and_attach_volume(svc, instance_id)
	fmt.Println("Waiting for ssh connect...")
	// start sshing there
	for {
	  if ssh_cmd(*instance_ip, "/bin/true") { break }
		time.Sleep(time.Second * 5)
	}
	// SCP the payload install file
	ssh_cp(*instance_ip, *payload_script_name)
	//panic("ok ready to DEVELOP")
	ssh_cmd(*instance_ip, "chmod +x " + *payload_script_name)
	// ssh there and run the payload
	fmt.Println("Running" + *payload_script_name)
  if ssh_cmd(*instance_ip, "sudo " + *payload_script_name + " /dev/xvdx") {
		fmt.Println("script success!")
		//ssh_cmd(*instance_ip, "sudo systemctl halt")// poweroff
	} else {
		fmt.Println("fail!")
		//fmt.Println("umounting /mnt")
		ssh_cmd(*instance_ip, "sudo umount /mnt")
		fmt.Println("Builder IP: " + *instance_ip)
		fmt.Println("Image build payload failed!  Aborting in-flight for inspection.")
		fail_reader := bufio.NewReader(os.Stdin)
	  fmt.Print("If you want to fix this manually. Press ENTER to continue the build (CTRL-C to abort):")
	  fail_reader.ReadString('\n')
	}
	// Pause for user inspection
	if *interactive_flag {
		reader := bufio.NewReader(os.Stdin)
		fmt.Println("Instance IP address: " + *instance_ip)
		fmt.Print("We're done in the chroot. Press ENTER to continue:")
		reader.ReadString('\n')
	}
	// detach the volume
	fmt.Println("Detaching volume...")
	if detach_volume_from_instance(svc, volume_id, instance_id) {
		fmt.Println("volume" + *volume_id + "detached")
	} else {
		panic("Fatal Error: failed to detach volume")
	}
	// wait for volume to be "available"
	wait_for_volume_available(svc, volume_id)
	// TODO: terminate builder instance?
	//# snapshot volume
	fmt.Println("Snapshot volume...")
	snapshot_id := snapshot_volume(svc, volume_id, image_title)
	//# wait for snapshot to be "completed"
	fmt.Println("Waiting for snapshot to be completed...")
	wait_snapshot(svc, snapshot_id)
	//# create AMI from snapshot
	ami_id := create_image(svc, snapshot_id, image_title)
	fmt.Println("AMI finished build: " + *ami_id)
  // cleanup un-necessary artifacts
	fmt.Println("Cleanup temporary volume...")
	cleanup(svc, volume_id)
	//# launch AMI
	fmt.Println("Launching test instance with this new AMI...")
	test_spot_req_id := req_spot(svc, *ami_id)
	test_instance_id := wait_for_spot_req_active(svc, test_spot_req_id)
	test_instance_ip := get_instance_ip(svc, test_instance_id)
	fmt.Println("test instance public ip: " + *test_instance_ip)
	
	fmt.Println("waiting for ssh...")
	for {
		if ssh_cmd(*test_instance_ip, "/bin/true") { break }
		time.Sleep(time.Second * 5)
	}	
	harvest_software_versions(svc, test_instance_ip)
	fmt.Println("AMI: " + *ami_id)
}
