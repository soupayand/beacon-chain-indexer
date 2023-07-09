package model

type BeaconChainData struct {
	Epoch               int64    `json:"epoch,omitempty"`
	ExecutionOptimistic bool     `json:"execution_optimistic,omitempty"`
	Finalized           bool     `json:"finalized,omitempty"`
	Data                SlotData `json:"data"`
}

type SlotData struct {
	Root      string     `json:"root,omitempty"`
	Canonical bool       `json:"canonical,omitempty"`
	Header    HeaderData `json:"header"`
	UnixTime  int64      `json:"unix_timestamp,omitempty"`
}

type HeaderData struct {
	Message   MessageData `json:"message"`
	Signature string      `json:"signature"`
}

type MessageData struct {
	Slot          string `json:"slot,omitempty"`
	ProposerIndex string `json:"proposer_index,omitempty"`
	ParentRoot    string `json:"parent_root,omitempty"`
	StateRoot     string `json:"state_root,omitempty"`
	BodyRoot      string `json:"body_root,omitempty"`
}

type Attestation struct {
	AggregationBits string             `json:"aggregation_bits"`
	Details         AttestationDetails `json:"data"`
	Signature       string             `json:"signature"`
}

type AttestationDetails struct {
	Slot            string `json:"slot"`
	Index           string `json:"index"`
	BeaconBlockRoot string `json:"beacon_block_root"`
	Source          Epoch  `json:"source"`
	Target          Epoch  `json:"target"`
}

type Epoch struct {
	Epoch string `json:"epoch"`
	Root  string `json:"root"`
}

type Committee struct {
	Index      string   `json:"index"`
	Slot       string   `json:"slot"`
	Validators []string `json:"validators"`
}

type Participation struct {
	ParticipationFactor float64 `json:"participation_factor"`
	MissedAttestations  int     `json:"missed_attestations"`
	ValidatorSetSize    int     `json:"validator_set_size"`
}
