package discovery

type S3Instance struct {
	// Id of the container running the S3 instance
	ContainerId string
	// Number of the S3 instance - beginning from 1
	InstanceNum int
	// Access key for the S3 instance, extracted from the container env
	AccessKey string
	// Secret key for the S3 instance, extracted from the container env
	SecretKey string
	// Container Network settings
	IpAddress string
	Hostname  string
	Port      string
}
