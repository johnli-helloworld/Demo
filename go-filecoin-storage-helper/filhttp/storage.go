package filhttp

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"strconv"
)

type StorageAPI HttpAPI

func (s *StorageAPI) Import(ctx context.Context, fr io.Reader) (cid string, err error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	formfile, err := writer.CreateFormFile("file", "")
	if _, err := io.Copy(formfile, fr); err != nil {
		return "", err
	}
	writer.Close()

	var out map[string]string
	err = Newhttp("").Request("client/import").
		Header("Content-Type", writer.FormDataContentType()).
		Body(body).
		Exec(ctx, &out)

	if err != nil {
		return "", err
	}
	if _, ok := out["/"]; ok {
		cid = out["/"]
	}
	return cid, nil
}

type StorageDealInfo struct {
	State       State             `json:"state"`
	Message     string            `json:"message"`
	ProposalCid map[string]string `json:"proposal_cid"`
	ProofInfo   ProofInfo         `json:"proofInfo"`
	Signature   string            `json:"signature"`
}

type ProofInfo struct {
	SectorID            uint64            `json:"sectorID"`
	CommD               []byte            `json:"commd"`
	CommR               []byte            `json:"commr"`
	CommRStar           []byte            `json:"comm_r_star"`
	CommitmentMessage   map[string]string `json:"commitment_message"`
	PieceInclusionProof []byte            `json:"piece_inclusion_proof"`
}

type DealInfo struct {
	State   string `json:"state"`
	Message string `json:"message"`
	DealId  string `json:"dealid"`
}

type State int

const (
	Unset = State(iota)
	Unknown
	Rejected
	Accepted
	Started
	Failed
	Staged
	Complete
)

func (s State) String() string {
	switch s {
	case Unset:
		return "unset"
	case Unknown:
		return "unknown"
	case Rejected:
		return "rejected"
	case Accepted:
		return "accepted"
	case Started:
		return "started"
	case Failed:
		return "failed"
	case Staged:
		return "staged"
	case Complete:
		return "complete"
	default:
		return fmt.Sprintf("<unrecognized %d>", s)
	}
}

func (s *StorageAPI) ProposeStorageDeal(ctx context.Context, miner string, cid string, askId string, time int64) (*DealInfo, error) {
	var out StorageDealInfo
	d := &DealInfo{}
	err := Newhttp("").Request("client/propose-storage-deal").
		Arguments(miner).
		Arguments(cid).
		Arguments(askId).
		Arguments(strconv.FormatInt(time, 10)).
		Exec(ctx, &out)
	if err != nil {
		fmt.Println("ProposeStorageDeal err:", err)
		return d, err
	}
	ProposalCid := out.ProposalCid
	if _, ok := ProposalCid["/"]; ok {
		d.DealId = ProposalCid["/"]
	}
	d.State = out.State.String()
	d.Message = out.Message
	fmt.Println("ProposeStorageDeal d:", d)
	return d, nil
}

//查询订单状态
func (s *StorageAPI) QueryStorageDeal(ctx context.Context, dealID string) (*DealInfo, error) {
	var out StorageDealInfo
	d := &DealInfo{}
	err := Newhttp("").Request("client/query-storage-deal", dealID).
		Exec(ctx, &out)
	if err != nil {
		return d, err
	}
	ProposalCid := out.ProposalCid
	if _, ok := ProposalCid["/"]; ok {
		d.DealId = ProposalCid["/"]
	}
	d.State = out.State.String()
	d.Message = out.Message
	return d, nil
}

//通过cid获取原数据库文件
func (s *StorageAPI) Cat(ctx context.Context, cid string) (io.Reader, error) {
	resp, err := Newhttp("").Request("client/cat", cid).
		Send(ctx)
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, resp.Error
	}
	defer resp.Close()

	b := new(bytes.Buffer)
	if _, err := io.Copy(b, resp.Output); err != nil {
		return nil, err
	}
	return b, nil
}
