// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.28.0
// 	protoc        v3.19.4
// source: job.proto

package protobuf

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type EnumFileType int32

const (
	EnumFileType_EFT_UNSPECIFIED EnumFileType = 0
	EnumFileType_EFT_TAR         EnumFileType = 1
)

// Enum value maps for EnumFileType.
var (
	EnumFileType_name = map[int32]string{
		0: "EFT_UNSPECIFIED",
		1: "EFT_TAR",
	}
	EnumFileType_value = map[string]int32{
		"EFT_UNSPECIFIED": 0,
		"EFT_TAR":         1,
	}
)

func (x EnumFileType) Enum() *EnumFileType {
	p := new(EnumFileType)
	*p = x
	return p
}

func (x EnumFileType) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (EnumFileType) Descriptor() protoreflect.EnumDescriptor {
	return file_job_proto_enumTypes[0].Descriptor()
}

func (EnumFileType) Type() protoreflect.EnumType {
	return &file_job_proto_enumTypes[0]
}

func (x EnumFileType) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use EnumFileType.Descriptor instead.
func (EnumFileType) EnumDescriptor() ([]byte, []int) {
	return file_job_proto_rawDescGZIP(), []int{0}
}

type EnumFileFetchMethod int32

const (
	EnumFileFetchMethod_EFFM_UNSPECIFIED EnumFileFetchMethod = 0
	EnumFileFetchMethod_EFFM_HTTP        EnumFileFetchMethod = 1
	EnumFileFetchMethod_EFFM_GRPC_STREAM EnumFileFetchMethod = 2
)

// Enum value maps for EnumFileFetchMethod.
var (
	EnumFileFetchMethod_name = map[int32]string{
		0: "EFFM_UNSPECIFIED",
		1: "EFFM_HTTP",
		2: "EFFM_GRPC_STREAM",
	}
	EnumFileFetchMethod_value = map[string]int32{
		"EFFM_UNSPECIFIED": 0,
		"EFFM_HTTP":        1,
		"EFFM_GRPC_STREAM": 2,
	}
)

func (x EnumFileFetchMethod) Enum() *EnumFileFetchMethod {
	p := new(EnumFileFetchMethod)
	*p = x
	return p
}

func (x EnumFileFetchMethod) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (EnumFileFetchMethod) Descriptor() protoreflect.EnumDescriptor {
	return file_job_proto_enumTypes[1].Descriptor()
}

func (EnumFileFetchMethod) Type() protoreflect.EnumType {
	return &file_job_proto_enumTypes[1]
}

func (x EnumFileFetchMethod) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use EnumFileFetchMethod.Descriptor instead.
func (EnumFileFetchMethod) EnumDescriptor() ([]byte, []int) {
	return file_job_proto_rawDescGZIP(), []int{1}
}

type EnumFileOperation int32

const (
	EnumFileOperation_EFO_UNSPECIFIED EnumFileOperation = 0
	EnumFileOperation_EFO_UNTAR       EnumFileOperation = 1
)

// Enum value maps for EnumFileOperation.
var (
	EnumFileOperation_name = map[int32]string{
		0: "EFO_UNSPECIFIED",
		1: "EFO_UNTAR",
	}
	EnumFileOperation_value = map[string]int32{
		"EFO_UNSPECIFIED": 0,
		"EFO_UNTAR":       1,
	}
)

func (x EnumFileOperation) Enum() *EnumFileOperation {
	p := new(EnumFileOperation)
	*p = x
	return p
}

func (x EnumFileOperation) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (EnumFileOperation) Descriptor() protoreflect.EnumDescriptor {
	return file_job_proto_enumTypes[2].Descriptor()
}

func (EnumFileOperation) Type() protoreflect.EnumType {
	return &file_job_proto_enumTypes[2]
}

func (x EnumFileOperation) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use EnumFileOperation.Descriptor instead.
func (EnumFileOperation) EnumDescriptor() ([]byte, []int) {
	return file_job_proto_rawDescGZIP(), []int{2}
}

type JobGetRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	UserId     string `protobuf:"bytes,1,opt,name=user_id,json=userId,proto3" json:"user_id,omitempty"`
	DeviceId   string `protobuf:"bytes,2,opt,name=device_id,json=deviceId,proto3" json:"device_id,omitempty"`
	DeviceInfo string `protobuf:"bytes,3,opt,name=device_info,json=deviceInfo,proto3" json:"device_info,omitempty"`
}

func (x *JobGetRequest) Reset() {
	*x = JobGetRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_job_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *JobGetRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*JobGetRequest) ProtoMessage() {}

func (x *JobGetRequest) ProtoReflect() protoreflect.Message {
	mi := &file_job_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use JobGetRequest.ProtoReflect.Descriptor instead.
func (*JobGetRequest) Descriptor() ([]byte, []int) {
	return file_job_proto_rawDescGZIP(), []int{0}
}

func (x *JobGetRequest) GetUserId() string {
	if x != nil {
		return x.UserId
	}
	return ""
}

func (x *JobGetRequest) GetDeviceId() string {
	if x != nil {
		return x.DeviceId
	}
	return ""
}

func (x *JobGetRequest) GetDeviceInfo() string {
	if x != nil {
		return x.DeviceInfo
	}
	return ""
}

type JobGetResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	JobId      string       `protobuf:"bytes,1,opt,name=job_id,json=jobId,proto3" json:"job_id,omitempty"`
	Image      *DockerImage `protobuf:"bytes,2,opt,name=image,proto3" json:"image,omitempty"`
	Cmds       []string     `protobuf:"bytes,3,rep,name=cmds,proto3" json:"cmds,omitempty"`
	VolumePath string       `protobuf:"bytes,4,opt,name=volume_path,json=volumePath,proto3" json:"volume_path,omitempty"`
	Files      []*File      `protobuf:"bytes,5,rep,name=files,proto3" json:"files,omitempty"`
	Outputs    []string     `protobuf:"bytes,6,rep,name=outputs,proto3" json:"outputs,omitempty"`
}

func (x *JobGetResponse) Reset() {
	*x = JobGetResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_job_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *JobGetResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*JobGetResponse) ProtoMessage() {}

func (x *JobGetResponse) ProtoReflect() protoreflect.Message {
	mi := &file_job_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use JobGetResponse.ProtoReflect.Descriptor instead.
func (*JobGetResponse) Descriptor() ([]byte, []int) {
	return file_job_proto_rawDescGZIP(), []int{1}
}

func (x *JobGetResponse) GetJobId() string {
	if x != nil {
		return x.JobId
	}
	return ""
}

func (x *JobGetResponse) GetImage() *DockerImage {
	if x != nil {
		return x.Image
	}
	return nil
}

func (x *JobGetResponse) GetCmds() []string {
	if x != nil {
		return x.Cmds
	}
	return nil
}

func (x *JobGetResponse) GetVolumePath() string {
	if x != nil {
		return x.VolumePath
	}
	return ""
}

func (x *JobGetResponse) GetFiles() []*File {
	if x != nil {
		return x.Files
	}
	return nil
}

func (x *JobGetResponse) GetOutputs() []string {
	if x != nil {
		return x.Outputs
	}
	return nil
}

type DockerImage struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Repository string `protobuf:"bytes,1,opt,name=repository,proto3" json:"repository,omitempty"`
	Digest     string `protobuf:"bytes,2,opt,name=digest,proto3" json:"digest,omitempty"`
	Tag        string `protobuf:"bytes,3,opt,name=tag,proto3" json:"tag,omitempty"`
	Uri        string `protobuf:"bytes,4,opt,name=uri,proto3" json:"uri,omitempty"`
}

func (x *DockerImage) Reset() {
	*x = DockerImage{}
	if protoimpl.UnsafeEnabled {
		mi := &file_job_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *DockerImage) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DockerImage) ProtoMessage() {}

func (x *DockerImage) ProtoReflect() protoreflect.Message {
	mi := &file_job_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DockerImage.ProtoReflect.Descriptor instead.
func (*DockerImage) Descriptor() ([]byte, []int) {
	return file_job_proto_rawDescGZIP(), []int{2}
}

func (x *DockerImage) GetRepository() string {
	if x != nil {
		return x.Repository
	}
	return ""
}

func (x *DockerImage) GetDigest() string {
	if x != nil {
		return x.Digest
	}
	return ""
}

func (x *DockerImage) GetTag() string {
	if x != nil {
		return x.Tag
	}
	return ""
}

func (x *DockerImage) GetUri() string {
	if x != nil {
		return x.Uri
	}
	return ""
}

type File struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Name       string            `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	Type       EnumFileType      `protobuf:"varint,2,opt,name=type,proto3,enum=protobuf.EnumFileType" json:"type,omitempty"`
	Preprocess EnumFileOperation `protobuf:"varint,3,opt,name=preprocess,proto3,enum=protobuf.EnumFileOperation" json:"preprocess,omitempty"`
	// Types that are assignable to Content:
	//	*File_Data
	//	*File_Remote
	Content isFile_Content `protobuf_oneof:"content"`
}

func (x *File) Reset() {
	*x = File{}
	if protoimpl.UnsafeEnabled {
		mi := &file_job_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *File) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*File) ProtoMessage() {}

func (x *File) ProtoReflect() protoreflect.Message {
	mi := &file_job_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use File.ProtoReflect.Descriptor instead.
func (*File) Descriptor() ([]byte, []int) {
	return file_job_proto_rawDescGZIP(), []int{3}
}

func (x *File) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *File) GetType() EnumFileType {
	if x != nil {
		return x.Type
	}
	return EnumFileType_EFT_UNSPECIFIED
}

func (x *File) GetPreprocess() EnumFileOperation {
	if x != nil {
		return x.Preprocess
	}
	return EnumFileOperation_EFO_UNSPECIFIED
}

func (m *File) GetContent() isFile_Content {
	if m != nil {
		return m.Content
	}
	return nil
}

func (x *File) GetData() []byte {
	if x, ok := x.GetContent().(*File_Data); ok {
		return x.Data
	}
	return nil
}

func (x *File) GetRemote() *FileUri {
	if x, ok := x.GetContent().(*File_Remote); ok {
		return x.Remote
	}
	return nil
}

type isFile_Content interface {
	isFile_Content()
}

type File_Data struct {
	Data []byte `protobuf:"bytes,4,opt,name=data,proto3,oneof"`
}

type File_Remote struct {
	Remote *FileUri `protobuf:"bytes,5,opt,name=remote,proto3,oneof"`
}

func (*File_Data) isFile_Content() {}

func (*File_Remote) isFile_Content() {}

type FileUri struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Uri         string              `protobuf:"bytes,1,opt,name=uri,proto3" json:"uri,omitempty"`
	FetchMethod EnumFileFetchMethod `protobuf:"varint,2,opt,name=fetch_method,json=fetchMethod,proto3,enum=protobuf.EnumFileFetchMethod" json:"fetch_method,omitempty"`
}

func (x *FileUri) Reset() {
	*x = FileUri{}
	if protoimpl.UnsafeEnabled {
		mi := &file_job_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *FileUri) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*FileUri) ProtoMessage() {}

func (x *FileUri) ProtoReflect() protoreflect.Message {
	mi := &file_job_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use FileUri.ProtoReflect.Descriptor instead.
func (*FileUri) Descriptor() ([]byte, []int) {
	return file_job_proto_rawDescGZIP(), []int{4}
}

func (x *FileUri) GetUri() string {
	if x != nil {
		return x.Uri
	}
	return ""
}

func (x *FileUri) GetFetchMethod() EnumFileFetchMethod {
	if x != nil {
		return x.FetchMethod
	}
	return EnumFileFetchMethod_EFFM_UNSPECIFIED
}

type JobPopulateRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	JobId    string `protobuf:"bytes,1,opt,name=job_id,json=jobId,proto3" json:"job_id,omitempty"`
	UserId   string `protobuf:"bytes,2,opt,name=user_id,json=userId,proto3" json:"user_id,omitempty"`
	DeviceId string `protobuf:"bytes,3,opt,name=device_id,json=deviceId,proto3" json:"device_id,omitempty"`
	Result   []byte `protobuf:"bytes,4,opt,name=result,proto3" json:"result,omitempty"`
	Status   int32  `protobuf:"varint,5,opt,name=status,proto3" json:"status,omitempty"`
}

func (x *JobPopulateRequest) Reset() {
	*x = JobPopulateRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_job_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *JobPopulateRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*JobPopulateRequest) ProtoMessage() {}

func (x *JobPopulateRequest) ProtoReflect() protoreflect.Message {
	mi := &file_job_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use JobPopulateRequest.ProtoReflect.Descriptor instead.
func (*JobPopulateRequest) Descriptor() ([]byte, []int) {
	return file_job_proto_rawDescGZIP(), []int{5}
}

func (x *JobPopulateRequest) GetJobId() string {
	if x != nil {
		return x.JobId
	}
	return ""
}

func (x *JobPopulateRequest) GetUserId() string {
	if x != nil {
		return x.UserId
	}
	return ""
}

func (x *JobPopulateRequest) GetDeviceId() string {
	if x != nil {
		return x.DeviceId
	}
	return ""
}

func (x *JobPopulateRequest) GetResult() []byte {
	if x != nil {
		return x.Result
	}
	return nil
}

func (x *JobPopulateRequest) GetStatus() int32 {
	if x != nil {
		return x.Status
	}
	return 0
}

type JobPopulateResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *JobPopulateResponse) Reset() {
	*x = JobPopulateResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_job_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *JobPopulateResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*JobPopulateResponse) ProtoMessage() {}

func (x *JobPopulateResponse) ProtoReflect() protoreflect.Message {
	mi := &file_job_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use JobPopulateResponse.ProtoReflect.Descriptor instead.
func (*JobPopulateResponse) Descriptor() ([]byte, []int) {
	return file_job_proto_rawDescGZIP(), []int{6}
}

var File_job_proto protoreflect.FileDescriptor

var file_job_proto_rawDesc = []byte{
	0x0a, 0x09, 0x6a, 0x6f, 0x62, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x08, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x62, 0x75, 0x66, 0x22, 0x66, 0x0a, 0x0d, 0x4a, 0x6f, 0x62, 0x47, 0x65, 0x74, 0x52,
	0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x17, 0x0a, 0x07, 0x75, 0x73, 0x65, 0x72, 0x5f, 0x69,
	0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x75, 0x73, 0x65, 0x72, 0x49, 0x64, 0x12,
	0x1b, 0x0a, 0x09, 0x64, 0x65, 0x76, 0x69, 0x63, 0x65, 0x5f, 0x69, 0x64, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x08, 0x64, 0x65, 0x76, 0x69, 0x63, 0x65, 0x49, 0x64, 0x12, 0x1f, 0x0a, 0x0b,
	0x64, 0x65, 0x76, 0x69, 0x63, 0x65, 0x5f, 0x69, 0x6e, 0x66, 0x6f, 0x18, 0x03, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x0a, 0x64, 0x65, 0x76, 0x69, 0x63, 0x65, 0x49, 0x6e, 0x66, 0x6f, 0x22, 0xc9, 0x01,
	0x0a, 0x0e, 0x4a, 0x6f, 0x62, 0x47, 0x65, 0x74, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65,
	0x12, 0x15, 0x0a, 0x06, 0x6a, 0x6f, 0x62, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x05, 0x6a, 0x6f, 0x62, 0x49, 0x64, 0x12, 0x2b, 0x0a, 0x05, 0x69, 0x6d, 0x61, 0x67, 0x65,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x15, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75,
	0x66, 0x2e, 0x44, 0x6f, 0x63, 0x6b, 0x65, 0x72, 0x49, 0x6d, 0x61, 0x67, 0x65, 0x52, 0x05, 0x69,
	0x6d, 0x61, 0x67, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x63, 0x6d, 0x64, 0x73, 0x18, 0x03, 0x20, 0x03,
	0x28, 0x09, 0x52, 0x04, 0x63, 0x6d, 0x64, 0x73, 0x12, 0x1f, 0x0a, 0x0b, 0x76, 0x6f, 0x6c, 0x75,
	0x6d, 0x65, 0x5f, 0x70, 0x61, 0x74, 0x68, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0a, 0x76,
	0x6f, 0x6c, 0x75, 0x6d, 0x65, 0x50, 0x61, 0x74, 0x68, 0x12, 0x24, 0x0a, 0x05, 0x66, 0x69, 0x6c,
	0x65, 0x73, 0x18, 0x05, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x0e, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x62, 0x75, 0x66, 0x2e, 0x46, 0x69, 0x6c, 0x65, 0x52, 0x05, 0x66, 0x69, 0x6c, 0x65, 0x73, 0x12,
	0x18, 0x0a, 0x07, 0x6f, 0x75, 0x74, 0x70, 0x75, 0x74, 0x73, 0x18, 0x06, 0x20, 0x03, 0x28, 0x09,
	0x52, 0x07, 0x6f, 0x75, 0x74, 0x70, 0x75, 0x74, 0x73, 0x22, 0x69, 0x0a, 0x0b, 0x44, 0x6f, 0x63,
	0x6b, 0x65, 0x72, 0x49, 0x6d, 0x61, 0x67, 0x65, 0x12, 0x1e, 0x0a, 0x0a, 0x72, 0x65, 0x70, 0x6f,
	0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0a, 0x72, 0x65,
	0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x12, 0x16, 0x0a, 0x06, 0x64, 0x69, 0x67, 0x65,
	0x73, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x64, 0x69, 0x67, 0x65, 0x73, 0x74,
	0x12, 0x10, 0x0a, 0x03, 0x74, 0x61, 0x67, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x74,
	0x61, 0x67, 0x12, 0x10, 0x0a, 0x03, 0x75, 0x72, 0x69, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x03, 0x75, 0x72, 0x69, 0x22, 0xd1, 0x01, 0x0a, 0x04, 0x46, 0x69, 0x6c, 0x65, 0x12, 0x12, 0x0a,
	0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d,
	0x65, 0x12, 0x2a, 0x0a, 0x04, 0x74, 0x79, 0x70, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0e, 0x32,
	0x16, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x6e, 0x75, 0x6d, 0x46,
	0x69, 0x6c, 0x65, 0x54, 0x79, 0x70, 0x65, 0x52, 0x04, 0x74, 0x79, 0x70, 0x65, 0x12, 0x3b, 0x0a,
	0x0a, 0x70, 0x72, 0x65, 0x70, 0x72, 0x6f, 0x63, 0x65, 0x73, 0x73, 0x18, 0x03, 0x20, 0x01, 0x28,
	0x0e, 0x32, 0x1b, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x6e, 0x75,
	0x6d, 0x46, 0x69, 0x6c, 0x65, 0x4f, 0x70, 0x65, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x0a,
	0x70, 0x72, 0x65, 0x70, 0x72, 0x6f, 0x63, 0x65, 0x73, 0x73, 0x12, 0x14, 0x0a, 0x04, 0x64, 0x61,
	0x74, 0x61, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0c, 0x48, 0x00, 0x52, 0x04, 0x64, 0x61, 0x74, 0x61,
	0x12, 0x2b, 0x0a, 0x06, 0x72, 0x65, 0x6d, 0x6f, 0x74, 0x65, 0x18, 0x05, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x11, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x46, 0x69, 0x6c, 0x65,
	0x55, 0x72, 0x69, 0x48, 0x00, 0x52, 0x06, 0x72, 0x65, 0x6d, 0x6f, 0x74, 0x65, 0x42, 0x09, 0x0a,
	0x07, 0x63, 0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74, 0x22, 0x5d, 0x0a, 0x07, 0x46, 0x69, 0x6c, 0x65,
	0x55, 0x72, 0x69, 0x12, 0x10, 0x0a, 0x03, 0x75, 0x72, 0x69, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x03, 0x75, 0x72, 0x69, 0x12, 0x40, 0x0a, 0x0c, 0x66, 0x65, 0x74, 0x63, 0x68, 0x5f, 0x6d,
	0x65, 0x74, 0x68, 0x6f, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x1d, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x6e, 0x75, 0x6d, 0x46, 0x69, 0x6c, 0x65, 0x46,
	0x65, 0x74, 0x63, 0x68, 0x4d, 0x65, 0x74, 0x68, 0x6f, 0x64, 0x52, 0x0b, 0x66, 0x65, 0x74, 0x63,
	0x68, 0x4d, 0x65, 0x74, 0x68, 0x6f, 0x64, 0x22, 0x91, 0x01, 0x0a, 0x12, 0x4a, 0x6f, 0x62, 0x50,
	0x6f, 0x70, 0x75, 0x6c, 0x61, 0x74, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x15,
	0x0a, 0x06, 0x6a, 0x6f, 0x62, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05,
	0x6a, 0x6f, 0x62, 0x49, 0x64, 0x12, 0x17, 0x0a, 0x07, 0x75, 0x73, 0x65, 0x72, 0x5f, 0x69, 0x64,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x75, 0x73, 0x65, 0x72, 0x49, 0x64, 0x12, 0x1b,
	0x0a, 0x09, 0x64, 0x65, 0x76, 0x69, 0x63, 0x65, 0x5f, 0x69, 0x64, 0x18, 0x03, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x08, 0x64, 0x65, 0x76, 0x69, 0x63, 0x65, 0x49, 0x64, 0x12, 0x16, 0x0a, 0x06, 0x72,
	0x65, 0x73, 0x75, 0x6c, 0x74, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x06, 0x72, 0x65, 0x73,
	0x75, 0x6c, 0x74, 0x12, 0x16, 0x0a, 0x06, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x18, 0x05, 0x20,
	0x01, 0x28, 0x05, 0x52, 0x06, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x22, 0x15, 0x0a, 0x13, 0x4a,
	0x6f, 0x62, 0x50, 0x6f, 0x70, 0x75, 0x6c, 0x61, 0x74, 0x65, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e,
	0x73, 0x65, 0x2a, 0x30, 0x0a, 0x0c, 0x45, 0x6e, 0x75, 0x6d, 0x46, 0x69, 0x6c, 0x65, 0x54, 0x79,
	0x70, 0x65, 0x12, 0x13, 0x0a, 0x0f, 0x45, 0x46, 0x54, 0x5f, 0x55, 0x4e, 0x53, 0x50, 0x45, 0x43,
	0x49, 0x46, 0x49, 0x45, 0x44, 0x10, 0x00, 0x12, 0x0b, 0x0a, 0x07, 0x45, 0x46, 0x54, 0x5f, 0x54,
	0x41, 0x52, 0x10, 0x01, 0x2a, 0x50, 0x0a, 0x13, 0x45, 0x6e, 0x75, 0x6d, 0x46, 0x69, 0x6c, 0x65,
	0x46, 0x65, 0x74, 0x63, 0x68, 0x4d, 0x65, 0x74, 0x68, 0x6f, 0x64, 0x12, 0x14, 0x0a, 0x10, 0x45,
	0x46, 0x46, 0x4d, 0x5f, 0x55, 0x4e, 0x53, 0x50, 0x45, 0x43, 0x49, 0x46, 0x49, 0x45, 0x44, 0x10,
	0x00, 0x12, 0x0d, 0x0a, 0x09, 0x45, 0x46, 0x46, 0x4d, 0x5f, 0x48, 0x54, 0x54, 0x50, 0x10, 0x01,
	0x12, 0x14, 0x0a, 0x10, 0x45, 0x46, 0x46, 0x4d, 0x5f, 0x47, 0x52, 0x50, 0x43, 0x5f, 0x53, 0x54,
	0x52, 0x45, 0x41, 0x4d, 0x10, 0x02, 0x2a, 0x37, 0x0a, 0x11, 0x45, 0x6e, 0x75, 0x6d, 0x46, 0x69,
	0x6c, 0x65, 0x4f, 0x70, 0x65, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x13, 0x0a, 0x0f, 0x45,
	0x46, 0x4f, 0x5f, 0x55, 0x4e, 0x53, 0x50, 0x45, 0x43, 0x49, 0x46, 0x49, 0x45, 0x44, 0x10, 0x00,
	0x12, 0x0d, 0x0a, 0x09, 0x45, 0x46, 0x4f, 0x5f, 0x55, 0x4e, 0x54, 0x41, 0x52, 0x10, 0x01, 0x42,
	0x29, 0x5a, 0x27, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x73, 0x61,
	0x74, 0x68, 0x2d, 0x72, 0x75, 0x6e, 0x2f, 0x65, 0x6e, 0x67, 0x69, 0x6e, 0x65, 0x2f, 0x70, 0x6b,
	0x67, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x33,
}

var (
	file_job_proto_rawDescOnce sync.Once
	file_job_proto_rawDescData = file_job_proto_rawDesc
)

func file_job_proto_rawDescGZIP() []byte {
	file_job_proto_rawDescOnce.Do(func() {
		file_job_proto_rawDescData = protoimpl.X.CompressGZIP(file_job_proto_rawDescData)
	})
	return file_job_proto_rawDescData
}

var file_job_proto_enumTypes = make([]protoimpl.EnumInfo, 3)
var file_job_proto_msgTypes = make([]protoimpl.MessageInfo, 7)
var file_job_proto_goTypes = []interface{}{
	(EnumFileType)(0),           // 0: protobuf.EnumFileType
	(EnumFileFetchMethod)(0),    // 1: protobuf.EnumFileFetchMethod
	(EnumFileOperation)(0),      // 2: protobuf.EnumFileOperation
	(*JobGetRequest)(nil),       // 3: protobuf.JobGetRequest
	(*JobGetResponse)(nil),      // 4: protobuf.JobGetResponse
	(*DockerImage)(nil),         // 5: protobuf.DockerImage
	(*File)(nil),                // 6: protobuf.File
	(*FileUri)(nil),             // 7: protobuf.FileUri
	(*JobPopulateRequest)(nil),  // 8: protobuf.JobPopulateRequest
	(*JobPopulateResponse)(nil), // 9: protobuf.JobPopulateResponse
}
var file_job_proto_depIdxs = []int32{
	5, // 0: protobuf.JobGetResponse.image:type_name -> protobuf.DockerImage
	6, // 1: protobuf.JobGetResponse.files:type_name -> protobuf.File
	0, // 2: protobuf.File.type:type_name -> protobuf.EnumFileType
	2, // 3: protobuf.File.preprocess:type_name -> protobuf.EnumFileOperation
	7, // 4: protobuf.File.remote:type_name -> protobuf.FileUri
	1, // 5: protobuf.FileUri.fetch_method:type_name -> protobuf.EnumFileFetchMethod
	6, // [6:6] is the sub-list for method output_type
	6, // [6:6] is the sub-list for method input_type
	6, // [6:6] is the sub-list for extension type_name
	6, // [6:6] is the sub-list for extension extendee
	0, // [0:6] is the sub-list for field type_name
}

func init() { file_job_proto_init() }
func file_job_proto_init() {
	if File_job_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_job_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*JobGetRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_job_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*JobGetResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_job_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*DockerImage); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_job_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*File); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_job_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*FileUri); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_job_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*JobPopulateRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_job_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*JobPopulateResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	file_job_proto_msgTypes[3].OneofWrappers = []interface{}{
		(*File_Data)(nil),
		(*File_Remote)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_job_proto_rawDesc,
			NumEnums:      3,
			NumMessages:   7,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_job_proto_goTypes,
		DependencyIndexes: file_job_proto_depIdxs,
		EnumInfos:         file_job_proto_enumTypes,
		MessageInfos:      file_job_proto_msgTypes,
	}.Build()
	File_job_proto = out.File
	file_job_proto_rawDesc = nil
	file_job_proto_goTypes = nil
	file_job_proto_depIdxs = nil
}
