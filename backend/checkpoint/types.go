package checkpoint

type CheckpointMetadata struct {
	Magic          uint32
	Version        int32
	CheckpointID   int64
	MasterStates   []MasterState
	OperatorStates []OperatorState
	Properties     *CheckpointProperties
	PropertiesRaw  []byte
}

type CheckpointSummary struct {
	Version        int32
	CheckpointID   int64
	NumOperators   int
	Operators      []OperatorSummary
	StateFilePaths []string
	InlineStrings  []string
	Properties     *CheckpointProperties
	PropertiesRaw  []byte
}

type OperatorSummary struct {
	Name           string
	UID            string
	OperatorID     [16]byte
	Parallelism    int32
	MaxParallelism int32
}

type MasterState struct {
	Version int32
	Name    string
	Payload []byte
}

type OperatorState struct {
	Name             string
	UID              string
	OperatorID       [16]byte
	Parallelism      int32
	MaxParallelism   int32
	CoordinatorState *StreamStateHandle
	SubtaskStates    []SubtaskState
	Finished         bool
}

type SubtaskState struct {
	Index                int32
	Finished             bool
	ManagedOperatorState *OperatorStateHandle
	RawOperatorState     *OperatorStateHandle
	ManagedKeyedState    KeyedStateHandle
	RawKeyedState        KeyedStateHandle
	InputChannelStates   []ChannelStateHandle
	OutputChannelStates  []ChannelStateHandle
}

type StreamHandleType byte

const (
	StreamHandleNull         StreamHandleType = 0
	StreamHandleByteStream   StreamHandleType = 1
	StreamHandleFile         StreamHandleType = 2
	StreamHandleRelative     StreamHandleType = 6
	StreamHandleSegmentFile  StreamHandleType = 15
	StreamHandleEmptySegment StreamHandleType = 16
)

type StreamStateHandle struct {
	Type      StreamHandleType
	Name      string
	Path      string
	Size      int64
	Data      []byte
	StartPos  int64
	Scope     int32
	LogicalID string
}

type OperatorStateHandle struct {
	Type                     OperatorStateHandleType
	StateNameToOffsets       map[string]OperatorStatePartition
	TaskOwnedDirectory       string
	SharedDirectory          string
	IsEmptyFileMergingHandle bool
	DelegateState            *StreamStateHandle
}

type OperatorStateHandleType byte

const (
	OperatorStateHandleNull          OperatorStateHandleType = 0
	OperatorStateHandlePartitionable OperatorStateHandleType = 4
	OperatorStateHandleFileMerging   OperatorStateHandleType = 17
)

type OperatorStatePartition struct {
	DistributionMode string
	Offsets          []int64
}

type KeyedStateHandleType byte

const (
	KeyedStateHandleNull                KeyedStateHandleType = 0
	KeyedStateHandleLegacy              KeyedStateHandleType = 3
	KeyedStateHandleIncrementalLegacy   KeyedStateHandleType = 5
	KeyedStateHandleSavepoint           KeyedStateHandleType = 7
	KeyedStateHandleChangelogLegacy     KeyedStateHandleType = 8
	KeyedStateHandleChangelogByte       KeyedStateHandleType = 9
	KeyedStateHandleChangelogFileLegacy KeyedStateHandleType = 10
	KeyedStateHandleIncrementalV2       KeyedStateHandleType = 11
	KeyedStateHandleKeyGroupsV2         KeyedStateHandleType = 12
	KeyedStateHandleChangelogFileV2     KeyedStateHandleType = 13
	KeyedStateHandleChangelogV2         KeyedStateHandleType = 14
)

type KeyedStateHandle interface {
	isKeyedStateHandle()
}

type KeyGroupsHandle struct {
	Type          KeyedStateHandleType
	StartKeyGroup int32
	NumKeyGroups  int32
	Offsets       []int64
	Delegate      *StreamStateHandle
	HandleID      string
}

func (KeyGroupsHandle) isKeyedStateHandle() {}

type IncrementalKeyGroupsHandle struct {
	Type             KeyedStateHandleType
	CheckpointID     int64
	BackendID        string
	StartKeyGroup    int32
	NumKeyGroups     int32
	CheckpointedSize int64
	MetaHandle       *StreamStateHandle
	SharedFiles      []HandleAndLocalPath
	PrivateFiles     []HandleAndLocalPath
	HandleID         string
}

func (IncrementalKeyGroupsHandle) isKeyedStateHandle() {}

type ChangelogStateHandle struct {
	Type              KeyedStateHandleType
	StartKeyGroup     int32
	NumKeyGroups      int32
	CheckpointedSize  int64
	Materialized      []KeyedStateHandle
	NonMaterialized   []KeyedStateHandle
	MaterializationID int64
	CheckpointID      int64
	HandleID          string
}

func (ChangelogStateHandle) isKeyedStateHandle() {}

type ChangelogByteIncrementHandle struct {
	Type          KeyedStateHandleType
	StartKeyGroup int32
	NumKeyGroups  int32
	FromSeq       int64
	ToSeq         int64
	Changes       []ChangelogStateChange
	HandleID      string
}

func (ChangelogByteIncrementHandle) isKeyedStateHandle() {}

type ChangelogStateChange struct {
	KeyGroup int32
	Data     []byte
}

type ChangelogFileIncrementHandle struct {
	Type             KeyedStateHandleType
	StartKeyGroup    int32
	NumKeyGroups     int32
	Offsets          []ChangelogStreamOffset
	StateSize        int64
	CheckpointedSize int64
	HandleID         string
	StorageID        string
}

func (ChangelogFileIncrementHandle) isKeyedStateHandle() {}

type ChangelogStreamOffset struct {
	Offset int64
	Handle *StreamStateHandle
}

type HandleAndLocalPath struct {
	LocalPath string
	Handle    *StreamStateHandle
}

type ChannelStateHandle struct {
	Type                  byte
	SubtaskIndex          int32
	GateOrPartition       int32
	ChannelOrSubpartition int32
	Offsets               []int64
	StateSize             int64
	Handle                *StreamStateHandle
	RawOffsets            []byte
}

type CheckpointProperties struct {
	CheckpointType  string
	SharingStrategy string
	Source          string
}
