# Blackbird AMI Builder
## Quickstart
### Requirements
* Go
* IAM user with permission for creating EC2 images
    
### Build    
```
git clone https://github.com/bbcx/blackbird-ami-builder
cd blackbird-ami-builder
go get
go build buildami.go
```

### Config
Copy the example configuration into place and edit it to replace with your account settings.
```
cp config.toml.example config.toml
```


### Run
```
export AWS_ACCESS_KEY_ID=xxx
export AWS_SECRET_ACCESS_KEY=xxx
./buildami
```
