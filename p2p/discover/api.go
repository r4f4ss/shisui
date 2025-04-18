package discover

import (
	"errors"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/p2p/discover/portalwire"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/holiman/uint256"
)

// DiscV5API json-rpc spec
// https://playground.open-rpc.org/?schemaUrl=https://raw.githubusercontent.com/ethereum/portal-network-specs/assembled-spec/jsonrpc/openrpc.json&uiSchema%5BappBar%5D%5Bui:splitView%5D=false&uiSchema%5BappBar%5D%5Bui:input%5D=false&uiSchema%5BappBar%5D%5Bui:examplesDropdown%5D=false
type DiscV5API struct {
	DiscV5 *UDPv5
}

func NewDiscV5API(discV5 *UDPv5) *DiscV5API {
	return &DiscV5API{discV5}
}

type PutContentResult struct {
	PeerCount     int  `json:"peerCount"`
	StoredLocally bool `json:"storedLocally"`
}

type NodeInfo struct {
	NodeId string `json:"nodeId"`
	Enr    string `json:"enr"`
	Ip     string `json:"ip"`
}

type RoutingTableInfo struct {
	Buckets     [][]string `json:"buckets"`
	LocalNodeId string     `json:"localNodeId"`
}

type DiscV5PongResp struct {
	EnrSeq        uint64 `json:"enrSeq"`
	RecipientIP   string `json:"recipientIP"`
	RecipientPort uint16 `json:"recipientPort"`
}

type PortalPongResp struct {
	EnrSeq     uint32 `json:"enrSeq"`
	DataRadius string `json:"dataRadius"`
}

type ContentInfo struct {
	Content     string `json:"content"`
	UtpTransfer bool   `json:"utpTransfer"`
}

type TraceContentResult struct {
	Content     string `json:"content"`
	UtpTransfer bool   `json:"utpTransfer"`
	Trace       Trace  `json:"trace"`
}

type Trace struct {
	Origin       string                   `json:"origin"`       // local node id
	TargetId     string                   `json:"targetId"`     // target content id
	ReceivedFrom string                   `json:"receivedFrom"` // the node id of which content from
	Responses    map[string]RespByNode    `json:"responses"`    // the node id and there response nodeIds
	Metadata     map[string]*NodeMetadata `json:"metadata"`     // node id and there metadata object
	StartedAtMs  int                      `json:"startedAtMs"`  // timestamp of the beginning of this request in milliseconds
	Cancelled    []string                 `json:"cancelled"`    // the node ids which are send but cancelled
}

type NodeMetadata struct {
	Enr      string `json:"enr"`
	Distance string `json:"distance"`
}

type RespByNode struct {
	DurationMs    int32    `json:"durationMs"`
	RespondedWith []string `json:"respondedWith"`
}

type Enrs struct {
	Enrs []string `json:"enrs"`
}

func (d *DiscV5API) NodeInfo() *NodeInfo {
	n := d.DiscV5.LocalNode().Node()

	return &NodeInfo{
		NodeId: "0x" + n.ID().String(),
		Enr:    n.String(),
		Ip:     n.IP().String(),
	}
}

func (d *DiscV5API) RoutingTableInfo() *RoutingTableInfo {
	n := d.DiscV5.LocalNode().Node()
	bucketNodes := d.DiscV5.RoutingTableInfo()

	return &RoutingTableInfo{
		Buckets:     bucketNodes,
		LocalNodeId: "0x" + n.ID().String(),
	}
}

func (d *DiscV5API) AddEnr(enr string) (bool, error) {
	n, err := enode.Parse(enode.ValidSchemes, enr)
	if err != nil {
		return false, err
	}

	// immediately add the node to the routing table
	d.DiscV5.tab.mutex.Lock()
	defer d.DiscV5.tab.mutex.Unlock()
	d.DiscV5.tab.handleAddNode(addNodeOp{node: n, isInbound: true, forceSetLive: true})
	return true, nil
}

func (d *DiscV5API) GetEnr(nodeId string) (bool, error) {
	id, err := enode.ParseID(nodeId)
	if err != nil {
		return false, err
	}
	n := d.DiscV5.tab.getNode(id)
	if n == nil {
		return false, errors.New("record not in local routing table")
	}

	return true, nil
}

func (d *DiscV5API) DeleteEnr(nodeId string) (bool, error) {
	id, err := enode.ParseID(nodeId)
	if err != nil {
		return false, err
	}

	n := d.DiscV5.tab.getNode(id)
	if n == nil {
		return false, errors.New("record not in local routing table")
	}

	d.DiscV5.tab.mutex.Lock()
	defer d.DiscV5.tab.mutex.Unlock()
	b := d.DiscV5.tab.bucket(n.ID())
	d.DiscV5.tab.deleteInBucket(b, n.ID())
	return true, nil
}

func (d *DiscV5API) LookupEnr(nodeId string) (string, error) {
	id, err := enode.ParseID(nodeId)
	if err != nil {
		return "", err
	}

	enr := d.DiscV5.ResolveNodeId(id)

	if enr == nil {
		return "", errors.New("record not found in DHT lookup")
	}

	return enr.String(), nil
}

func (d *DiscV5API) Ping(enr string) (*DiscV5PongResp, error) {
	n, err := enode.Parse(enode.ValidSchemes, enr)
	if err != nil {
		return nil, err
	}

	pong, err := d.DiscV5.pingInner(n)
	if err != nil {
		return nil, err
	}

	return &DiscV5PongResp{
		EnrSeq:        pong.ENRSeq,
		RecipientIP:   pong.ToIP.String(),
		RecipientPort: pong.ToPort,
	}, nil
}

func (d *DiscV5API) FindNodes(enr string, distances []uint) ([]string, error) {
	n, err := enode.Parse(enode.ValidSchemes, enr)
	if err != nil {
		return nil, err
	}
	findNodes, err := d.DiscV5.findnode(n, distances)
	if err != nil {
		return nil, err
	}

	enrs := make([]string, 0, len(findNodes))
	for _, r := range findNodes {
		enrs = append(enrs, r.String())
	}

	return enrs, nil
}

func (d *DiscV5API) TalkReq(enr string, protocol string, payload string) (string, error) {
	n, err := enode.Parse(enode.ValidSchemes, enr)
	if err != nil {
		return "", err
	}

	req, err := hexutil.Decode(payload)
	if err != nil {
		return "", err
	}

	talkResp, err := d.DiscV5.TalkRequest(n, protocol, req)
	if err != nil {
		return "", err
	}
	return hexutil.Encode(talkResp), nil
}

func (d *DiscV5API) RecursiveFindNodes(nodeId string) ([]string, error) {
	findNodes := d.DiscV5.Lookup(enode.HexID(nodeId))

	enrs := make([]string, 0, len(findNodes))
	for _, r := range findNodes {
		enrs = append(enrs, r.String())
	}

	return enrs, nil
}

type PortalProtocolAPI struct {
	portalProtocol *PortalProtocol
}

func NewPortalAPI(portalProtocol *PortalProtocol) *PortalProtocolAPI {
	return &PortalProtocolAPI{
		portalProtocol: portalProtocol,
	}
}

func (p *PortalProtocolAPI) NodeInfo() *NodeInfo {
	n := p.portalProtocol.localNode.Node()

	return &NodeInfo{
		NodeId: n.ID().String(),
		Enr:    n.String(),
		Ip:     n.IP().String(),
	}
}

func (p *PortalProtocolAPI) RoutingTableInfo() *RoutingTableInfo {
	n := p.portalProtocol.localNode.Node()
	bucketNodes := p.portalProtocol.RoutingTableInfo()

	return &RoutingTableInfo{
		Buckets:     bucketNodes,
		LocalNodeId: "0x" + n.ID().String(),
	}
}

func (p *PortalProtocolAPI) AddEnr(enr string) (bool, error) {
	p.portalProtocol.Log.Debug("serving AddEnr", "enr", enr)
	n, err := enode.ParseForAddEnr(enode.ValidSchemes, enr)
	if err != nil {
		return false, err
	}
	p.portalProtocol.AddEnr(n)
	return true, nil
}

func (p *PortalProtocolAPI) AddEnrs(enrs []string) bool {
	// Note: unspecified RPC, but useful for our local testnet test
	for _, enr := range enrs {
		n, err := enode.ParseForAddEnr(enode.ValidSchemes, enr)
		if err != nil {
			continue
		}
		p.portalProtocol.AddEnr(n)
	}

	return true
}

func (p *PortalProtocolAPI) GetEnr(nodeId string) (string, error) {
	id, err := enode.ParseID(nodeId)
	if err != nil {
		return "", err
	}

	if id == p.portalProtocol.localNode.Node().ID() {
		return p.portalProtocol.localNode.Node().String(), nil
	}

	n := p.portalProtocol.table.getNode(id)
	if n == nil {
		return "", errors.New("record not in local routing table")
	}

	return n.String(), nil
}

func (p *PortalProtocolAPI) DeleteEnr(nodeId string) (bool, error) {
	id, err := enode.ParseID(nodeId)
	if err != nil {
		return false, err
	}

	n := p.portalProtocol.table.getNode(id)
	if n == nil {
		return false, nil
	}

	p.portalProtocol.table.mutex.Lock()
	defer p.portalProtocol.table.mutex.Unlock()
	b := p.portalProtocol.table.bucket(n.ID())
	p.portalProtocol.table.deleteInBucket(b, n.ID())
	return true, nil
}

func (p *PortalProtocolAPI) LookupEnr(nodeId string) (string, error) {
	id, err := enode.ParseID(nodeId)
	if err != nil {
		return "", err
	}

	enr := p.portalProtocol.ResolveNodeId(id)

	if enr == nil {
		return "", errors.New("record not found in DHT lookup")
	}

	return enr.String(), nil
}

func (p *PortalProtocolAPI) Ping(enr string) (*PortalPongResp, error) {
	n, err := enode.Parse(enode.ValidSchemes, enr)
	if err != nil {
		return nil, err
	}

	pong, err := p.portalProtocol.pingInner(n)
	if err != nil {
		return nil, err
	}

	customPayload := &portalwire.PingPongCustomData{}
	err = customPayload.UnmarshalSSZ(pong.CustomPayload)
	if err != nil {
		return nil, err
	}

	nodeRadius := new(uint256.Int)
	err = nodeRadius.UnmarshalSSZ(customPayload.Radius)
	if err != nil {
		return nil, err
	}

	return &PortalPongResp{
		EnrSeq:     uint32(pong.EnrSeq),
		DataRadius: nodeRadius.Hex(),
	}, nil
}

func (p *PortalProtocolAPI) FindNodes(enr string, distances []uint) ([]string, error) {
	n, err := enode.Parse(enode.ValidSchemes, enr)
	if err != nil {
		return nil, err
	}
	findNodes, err := p.portalProtocol.findNodes(n, distances)
	if err != nil {
		return nil, err
	}

	enrs := make([]string, 0, len(findNodes))
	for _, r := range findNodes {
		enrs = append(enrs, r.String())
	}

	return enrs, nil
}

func (p *PortalProtocolAPI) FindContent(enr string, contentKey string) (interface{}, error) {
	n, err := enode.Parse(enode.ValidSchemes, enr)
	if err != nil {
		return nil, err
	}

	contentKeyBytes, err := hexutil.Decode(contentKey)
	if err != nil {
		return nil, err
	}

	flag, findContent, err := p.portalProtocol.findContent(n, contentKeyBytes)
	if err != nil {
		return nil, err
	}

	switch flag {
	case portalwire.ContentRawSelector:
		contentInfo := &ContentInfo{
			Content:     hexutil.Encode(findContent.([]byte)),
			UtpTransfer: false,
		}
		p.portalProtocol.Log.Trace("FindContent", "contentInfo", contentInfo)
		return contentInfo, nil
	case portalwire.ContentConnIdSelector:
		contentInfo := &ContentInfo{
			Content:     hexutil.Encode(findContent.([]byte)),
			UtpTransfer: true,
		}
		p.portalProtocol.Log.Trace("FindContent", "contentInfo", contentInfo)
		return contentInfo, nil
	default:
		enrs := make([]string, 0)
		for _, r := range findContent.([]*enode.Node) {
			enrs = append(enrs, r.String())
		}

		p.portalProtocol.Log.Trace("FindContent", "enrs", enrs)
		return &Enrs{
			Enrs: enrs,
		}, nil
	}
}

func (p *PortalProtocolAPI) Offer(enr string, contentItems [][2]string) (string, error) {
	n, err := enode.Parse(enode.ValidSchemes, enr)
	if err != nil {
		return "", err
	}

	entries := make([]*ContentEntry, 0, len(contentItems))
	for _, contentItem := range contentItems {
		contentKey, err := hexutil.Decode(contentItem[0])
		if err != nil {
			return "", err
		}
		contentValue, err := hexutil.Decode(contentItem[1])
		if err != nil {
			return "", err
		}
		contentEntry := &ContentEntry{
			ContentKey: contentKey,
			Content:    contentValue,
		}
		entries = append(entries, contentEntry)
	}

	transientOfferRequest := &TransientOfferRequest{
		Contents: entries,
	}

	offerReq := &OfferRequest{
		Kind:    TransientOfferRequestKind,
		Request: transientOfferRequest,
	}
	accept, err := p.portalProtocol.offer(n, offerReq)
	if err != nil {
		return "", err
	}

	return hexutil.Encode(accept), nil
}

func (p *PortalProtocolAPI) RecursiveFindNodes(nodeId string) ([]string, error) {
	findNodes := p.portalProtocol.Lookup(enode.HexID(nodeId))

	enrs := make([]string, 0, len(findNodes))
	for _, r := range findNodes {
		enrs = append(enrs, r.String())
	}

	return enrs, nil
}

func (p *PortalProtocolAPI) RecursiveFindContent(contentKeyHex string) (*ContentInfo, error) {
	contentKey, err := hexutil.Decode(contentKeyHex)
	if err != nil {
		return nil, err
	}
	contentId := p.portalProtocol.toContentId(contentKey)

	data, err := p.portalProtocol.Get(contentKey, contentId)
	if err == nil {
		return &ContentInfo{
			Content:     hexutil.Encode(data),
			UtpTransfer: false,
		}, err
	}
	p.portalProtocol.Log.Warn("find content err", "contextKey", hexutil.Encode(contentKey), "err", err)

	content, utpTransfer, err := p.portalProtocol.ContentLookup(contentKey, contentId)

	if err != nil {
		return nil, err
	}

	return &ContentInfo{
		Content:     hexutil.Encode(content),
		UtpTransfer: utpTransfer,
	}, err
}

func (p *PortalProtocolAPI) LocalContent(contentKeyHex string) (string, error) {
	contentKey, err := hexutil.Decode(contentKeyHex)
	if err != nil {
		return "", err
	}
	contentId := p.portalProtocol.ToContentId(contentKey)
	content, err := p.portalProtocol.Get(contentKey, contentId)

	if err != nil {
		return "", err
	}
	return hexutil.Encode(content), nil
}

func (p *PortalProtocolAPI) Store(contentKeyHex string, contextHex string) (bool, error) {
	contentKey, err := hexutil.Decode(contentKeyHex)
	if err != nil {
		return false, err
	}
	contentId := p.portalProtocol.ToContentId(contentKey)
	if !p.portalProtocol.InRange(contentId) {
		return false, nil
	}
	content, err := hexutil.Decode(contextHex)
	if err != nil {
		return false, err
	}
	err = p.portalProtocol.Put(contentKey, contentId, content)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (p *PortalProtocolAPI) Gossip(contentKeyHex, contentHex string) (int, error) {
	contentKey, err := hexutil.Decode(contentKeyHex)
	if err != nil {
		return 0, err
	}
	content, err := hexutil.Decode(contentHex)
	if err != nil {
		return 0, err
	}
	id := p.portalProtocol.Self().ID()
	return p.portalProtocol.Gossip(&id, [][]byte{contentKey}, [][]byte{content})
}

func (p *PortalProtocolAPI) PutContent(contentKeyHex, contentHex string) (*PutContentResult, error) {
	contentKey, err := hexutil.Decode(contentKeyHex)
	if err != nil {
		return nil, err
	}
	content, err := hexutil.Decode(contentHex)
	if err != nil {
		return nil, err
	}
	shouldStore, err := p.portalProtocol.ShouldStore(contentKey, content)
	if err != nil {
		return nil, err
	}
	id := p.portalProtocol.Self().ID()
	num, err := p.portalProtocol.Gossip(&id, [][]byte{contentKey}, [][]byte{content})
	if err != nil {
		return nil, err
	}
	return &PutContentResult{
		PeerCount:     num,
		StoredLocally: shouldStore,
	}, nil
}

func (p *PortalProtocolAPI) TraceRecursiveFindContent(contentKeyHex string) (*TraceContentResult, error) {
	contentKey, err := hexutil.Decode(contentKeyHex)
	if err != nil {
		return nil, err
	}
	contentId := p.portalProtocol.toContentId(contentKey)
	return p.portalProtocol.TraceContentLookup(contentKey, contentId)
}
