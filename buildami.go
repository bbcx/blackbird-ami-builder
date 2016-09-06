package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/jeremyd/easyssh"
	"github.com/spf13/viper"
)

func launchInstance(client *ec2.EC2, ami *string) *string {
	launchParams := &ec2.RunInstancesInput{
		ImageId:  ami,
		MaxCount: aws.Int64(1),
		MinCount: aws.Int64(1),
		//EbsOptimized:          aws.Bool(true),
		//IamInstanceProfile: &ec2.IamInstanceProfileSpecification{
		//Arn: aws.String(instanceProfileArn),
		//	Name: aws.String("k8s-master" + viper.GetString("cluster-name")),
		//},
		InstanceInitiatedShutdownBehavior: aws.String("terminate"),
		InstanceType:                      aws.String("m4.large"),
		KeyName:                           aws.String(viper.GetString("build.ssh_key_name")),
		SecurityGroupIds: []*string{
			aws.String(viper.GetString("build.security_group_id")),
		},
		SubnetId: aws.String(viper.GetString("build.subnet_id")),
		//UserData: aws.String(userData),
	}
	resp, err := client.RunInstances(launchParams)
	if err != nil {
		fmt.Println("Error launching instance.")
		fmt.Println(err)
		return nil
	}
	return resp.Instances[0].InstanceId
}

func get_instance_ip(client *ec2.EC2, id *string, count int64) (ip *string) {
	params := &ec2.DescribeInstancesInput{
		InstanceIds: []*string{id}}

	resp, err := client.DescribeInstances(params)
	count--
	returnMe := resp.Reservations[0].Instances[0].PublicIpAddress
	if err != nil || returnMe == nil {
		fmt.Print(".")
		if count == 0 {
			fmt.Println("retry count exceeded getting instance ip. exiting.")
			os.Exit(1)
		}
		time.Sleep(time.Second)
		return get_instance_ip(client, id, count)
	}
	//fmt.Println(resp)
	return returnMe
}

func write_state(instanceID *string) {
	d1 := []byte(*instanceID)
	err := ioutil.WriteFile(".buildami.state", d1, 0644)
	if err != nil {
		fmt.Println("WARNING: error occured writing to .buildami.state file")
	}
}

func create_and_attach_volume(client *ec2.EC2, instance_id *string) (volume_id *string) {
	// DESCRIBE INSTANCE TO GET METADATA
	iparams := &ec2.DescribeInstancesInput{
		InstanceIds: []*string{instance_id}}
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
			VolumeId:   save_volume_id}
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

func ssh_cmd(instance_ip string, command string) (output string, success bool) {
	ssh := &easyssh.MakeConfig{
		User:   "root",
		Server: instance_ip,
		// Optional key or Password without either we try to contact your agent SOCKET
		Key:  viper.GetString("build.ssh_key_path"),
		Port: "22",
	}
	// Call Run method with command you want to run on remote server.
	output, err := ssh.Run(command)
	// Handle errors
	if err != nil {
		fmt.Println("Can't run remote command: " + err.Error())
		return "", false
	}
	return output, true
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
		InstanceId: instance_id}
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
		SnapshotIds: []*string{snapshot_id}}
	err := client.WaitUntilSnapshotCompleted(params)
	if err != nil {
		fmt.Println(err.Error())
	}
}

func create_image(client *ec2.EC2, snapshot_id *string, title string) (image_id *string) {
	image_size := int64(viper.GetInt("build.image_size"))
	params := &ec2.RegisterImageInput{
		Name: aws.String(title), // Required
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
			{
				DeviceName:  aws.String("/dev/sdb"),
				VirtualName: aws.String("ephemeral0"),
			},
			{
				DeviceName:  aws.String("/dev/sdc"),
				VirtualName: aws.String("ephemeral0"),
			},
			{
				DeviceName:  aws.String("/dev/sdd"),
				VirtualName: aws.String("ephemeral0"),
			},
			{
				DeviceName:  aws.String("/dev/sde"),
				VirtualName: aws.String("ephemeral0"),
			},
		},
		Architecture:   aws.String(ec2.ArchitectureValuesX8664),
		Description:    aws.String(title),
		RootDeviceName: aws.String("/dev/sda1"),
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

func waitAndMakePublic(amiID *string, regionClient *ec2.EC2, wg *sync.WaitGroup) {
	defer wg.Done()
	// Wait for image to be available
	fmt.Println("waiting for " + *amiID + " to be available...")
	describeImagesInput := &ec2.DescribeImagesInput{
		ImageIds: []*string{amiID},
	}
	if err := regionClient.WaitUntilImageAvailable(describeImagesInput); err !=
		nil {
		fmt.Println(err)
	}

	fmt.Println("making image public " + *amiID)
	// Make public
	publicParams := &ec2.ModifyImageAttributeInput{
		ImageId:   amiID, // Required
		Attribute: aws.String("launchPermission"),
		LaunchPermission: &ec2.LaunchPermissionModifications{
			Add: []*ec2.LaunchPermission{
				{
					Group: aws.String("all"),
				},
			},
		},
	}
	_, errPub := regionClient.ModifyImageAttribute(publicParams)
	if errPub != nil {
		fmt.Println("error modifying " + *amiID + " to be public")
		fmt.Println(errPub)
	}
}

func publishRegions(svc *ec2.EC2, amiID string, imageTitle string) {
	fmt.Println("publishing to regions...")
	regionList := strings.Split(viper.GetString("publish.regions"), ",")
	fmt.Println(regionList)
	var wg sync.WaitGroup

	// Make public the default region image.
	if viper.GetString("publish.make_public") == "true" {
		wg.Add(1)
		go waitAndMakePublic(&amiID, svc, &wg)
	}

	// Copy and make public for all specified regions.
	for destRegion := range regionList {
		// create new ec2 client for this destination region
		regionClient := ec2.New(session.New(), &aws.Config{Region: aws.String(regionList[destRegion])})
		// copy to dest region
		copyParams := &ec2.CopyImageInput{
			Description:   aws.String(imageTitle),
			Name:          aws.String(imageTitle),
			SourceImageId: aws.String(amiID),
			SourceRegion:  aws.String(viper.GetString("build.region")),
		}
		respCopy, errCopy := regionClient.CopyImage(copyParams)
		if errCopy != nil {
			fmt.Println("something went wrong copying to dest region: " + regionList[destRegion])
			fmt.Println(errCopy)
		}
		fmt.Println(regionList[destRegion] + "," + *respCopy.ImageId)
		wg.Add(1)
		if viper.GetString("publish.make_public") == "true" {
			go waitAndMakePublic(respCopy.ImageId, regionClient, &wg)
		}
		recordManifestID(respCopy.ImageId, regionList[destRegion])
	}

	if viper.GetString("publish.make_public") == "true" {
		wg.Wait()
	}
	fmt.Println("all regions copied")
}

func recordManifestID(amiID *string, region string) {
	// Record test output to file
	theBytes := []byte(*amiID)
	outPath := path.Join(viper.GetString("publish.manifest_dir"), region)
	err := ioutil.WriteFile(outPath, theBytes, 0644)

	if err != nil {
		fmt.Println("WARNING: error occured writing to region manifest file" + outPath)
		fmt.Println(err)
	}
}

func main() {
	// Viper configuration engine
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	viper.SetEnvPrefix("BB")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
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
	instanceID := aws.String("")
	dat, err := ioutil.ReadFile(".buildami.state")
	if *use_existing_builder && err == nil {
		fmt.Println("Using existing builder from " + string(dat))
		instanceID = aws.String(string(dat))
	} else {
		fmt.Println("Launching Instance")
		instanceID = launchInstance(svc, aws.String(viper.GetString("build.ami_id")))
		write_state(instanceID)
	}
	//spot_req_id := aws.String("sir-03hfzd70")
	//fmt.Println("Waiting for spot instance to become active...")
	//instance_id := wait_for_spot_req_active(svc, spot_req_id)
	// get it's IP address
	fmt.Println("Getting instance ip address...")
	// TODO: there's a race condition here where instance IP isn't available yet..
	instance_ip := get_instance_ip(svc, instanceID, 30)
	fmt.Println("Instance IP address: " + *instance_ip)
	// create and attach
	fmt.Println("Creating and attaching new volume...")
	volume_id := create_and_attach_volume(svc, instanceID)
	fmt.Println("Waiting for ssh connect...")
	// start sshing there
	for {
		output, success := ssh_cmd(*instance_ip, "/bin/true")
		if success {
			break
		}
		fmt.Println(output)

		time.Sleep(time.Second * 5)
	}
	// SCP the payload install file
	ssh_cp(*instance_ip, *payload_script_name)
	//panic("ok ready to DEVELOP")
	ssh_cmd(*instance_ip, "chmod +x "+*payload_script_name)
	// ssh there and run the payload
	fmt.Println("Running" + *payload_script_name)
	chrootOutput, success := ssh_cmd(*instance_ip, *payload_script_name+" /dev/xvdx")
	// Record chroot output to file
	chrootBytes := []byte(chrootOutput)
	chrootOutputPath := path.Join(viper.GetString("publish.manifest_dir"), viper.GetString("build.region")+"_chroot.log")
	errChrootFile := ioutil.WriteFile(chrootOutputPath, chrootBytes, 0644)
	if errChrootFile != nil {
		fmt.Println("WARNING: error occured writing to chroot.log file")
		fmt.Println(errChrootFile)
	}
	if success {
		fmt.Println("script success!")
		//ssh_cmd(*instance_ip, "sudo systemctl halt")// poweroff
	} else {
		fmt.Println("fail!")
		//fmt.Println("umounting /mnt")
		ssh_cmd(*instance_ip, "sudo umount /mnt")
		fmt.Println("Builder IP: " + *instance_ip)
		if *interactive_flag {
			fmt.Println("Image build payload failed!  Aborting in-flight for inspection.")
			fail_reader := bufio.NewReader(os.Stdin)
			fmt.Print("If you want to fix this manually. Press ENTER to continue the build (CTRL-C to abort):")
			fail_reader.ReadString('\n')
		} else {
			fmt.Println("Aborting build due to chroot failure.")
			os.Exit(1)
		}
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
	if detach_volume_from_instance(svc, volume_id, instanceID) {
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

	//test_spot_req_id := req_spot(svc, *ami_id)
	//test_instance_id := wait_for_spot_req_active(svc, test_spot_req_id)
	//test_instance_ip := get_instance_ip(svc, test_instance_id)

	testInstanceID := launchInstance(svc, ami_id)
	time.Sleep(time.Second * 10)
	testInstanceIP := get_instance_ip(svc, testInstanceID, 0)

	fmt.Println("test instance public ip: " + *testInstanceIP)

	fmt.Println("waiting for ssh...")
	for {
		_, success := ssh_cmd(*testInstanceIP, "/bin/true")
		if success {
			break
		}
		time.Sleep(time.Second * 5)
	}

	// Run test harness
	ssh_cp(*testInstanceIP, "./testarch.sh")
	ssh_cmd(*testInstanceIP, "chmod +x testarch.sh")
	testOutput, testSuccess := ssh_cmd(*testInstanceIP, "./testarch.sh")

	// Harvest Software Versions from test instance
	ssh_cp(*testInstanceIP, "./harvestversions.sh")
	ssh_cmd(*testInstanceIP, "chmod +x harvestversions.sh")
	versionOutput, _ := ssh_cmd(*testInstanceIP, "./harvestversions.sh")
	versionBytes := []byte(versionOutput)
	versionOutputDir := path.Join(viper.GetString("publish.manifest_dir"), "versions.md")
	errVersion := ioutil.WriteFile(versionOutputDir, versionBytes, 0644)

	if errVersion != nil {
		fmt.Println("WARNING: error occured writing to versions.md file")
		fmt.Println(errVersion)
	}

	// Print test output to screen
	fmt.Println(testOutput)
	// Record test output to file
	testBytes := []byte(testOutput)
	testOutputPath := path.Join(viper.GetString("publish.manifest_dir"), *ami_id+"_"+viper.GetString("build.region")+"_testoutput.log")
	errTest := ioutil.WriteFile(testOutputPath, testBytes, 0644)

	if errTest != nil {
		fmt.Println("WARNING: error occured writing to testoutput.log file")
		fmt.Println(errTest)
	}

	fmt.Println("AMI: " + *ami_id)
	recordManifestID(ami_id, viper.GetString("build.region"))

	if testSuccess {
		if *interactive_flag {
			continueReader := bufio.NewReader(os.Stdin)
			fmt.Print("Testing complete ^^ Pausing for user input before publishing. Press ENTER to continue, CTRL-C to abort.")
			continueReader.ReadString('\n')
		}
		// Terminate test instance
		termParams := &ec2.TerminateInstancesInput{
			InstanceIds: []*string{testInstanceID},
		}
		_, termErr := svc.TerminateInstances(termParams)
		if termErr != nil {
			fmt.Println("error occured terminating test instance.")
			fmt.Println(termErr)
		} else {
			fmt.Println("test instance terminated.")
		}

		// Terminate build instance
		termBuildInstance := &ec2.TerminateInstancesInput{
			InstanceIds: []*string{instanceID},
		}
		_, termBuildError := svc.TerminateInstances(termBuildInstance)
		if termBuildError != nil {
			fmt.Println("error occured terminating bulid instance.")
			fmt.Println(termBuildInstance)
		} else {
			fmt.Println("build instance terminated.")
		}

		publishRegions(svc, *ami_id, image_title)
	} else {
		fmt.Println("tests failed ^^ aborting publishing.")
		os.Exit(1)
	}

}
