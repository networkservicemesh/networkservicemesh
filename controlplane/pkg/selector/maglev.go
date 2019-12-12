package selector

import (
	//"fmt"
	//"math"
	//"math/rand"
	"sync"
	"strconv"
	"github.com/sirupsen/logrus"
	//"google.golang.org/grpc"
	//"golang.org/x/net/context"
	//"github.com/golang/protobuf/ptypes/empty"
	//"github.com/golang/protobuf/proto"

	"github.com/dchest/siphash"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/crossconnect"
	
	//"k8s.io/client-go/rest"
	//"k8s.io/client-go/tools/clientcmd"
	//"github.com/skydive-project/skydive/logging"

)


type maglevSelector struct {
	sync.Mutex
	nseNbr uint64                             // Number of NSE pods (i.e., backend in the pool)
	Lk_Size uint64                             // size of the lookup table (number of requests)
	permutation [][]uint64                         // used to compute hash value per pod
	
	LookupTable map[uint64]int                  // Lookup table considering ReqId instead of index
	nseList []*registry.NetworkServiceEndpoint // list of nse candidates

	maglev map[string]uint64
}

// Batch variable, to check whether the request is the first on the batch or not
var FirstRequestInBatch bool = true

// Number of requests in the batch, corresponding to the size of Lookup table
var BatchSize uint64 = 2 // to be varied later

// Should be a global variable in Model
var RequestIdPerNetworkService map[string]map[string]uint64

// Lookup table for each NS to save decision
var LookupTablePerNetworkService map[string]map[uint64]int


var MetricsPerEndpoint = map[string]map[string]*crossconnect.Metrics{}
//var CurrentServerMetrics = crossconnect.Metrics{}


var informerStopper chan struct{}


func NewmaglevSelector() Selector {
	CreateMaglevBatchMaps()
	return &maglevSelector{
		LookupTable: make(map[uint64]int),
		maglev: make(map[string]uint64),
	}
}

// to be set at the begining, or for each incoming batch !!! (in the model for example, but only once)
func CreateMaglevBatchMaps() {
	RequestIdPerNetworkService = make (map[string]map[string]uint64)
	LookupTablePerNetworkService = make (map[string]map[uint64]int)
}


func (mg *maglevSelector) CreateMaglev(nseCandidates []*registry.NetworkServiceEndpoint) error {

	//logrus.Infof("create Magglev for len(nse) %d", len(nseCandidates))
	
	mg.Lk_Size =  BatchSize 
	n := uint64(len(nseCandidates))

	//logrus.Infof("set nseList to nseCandidates ")
	mg.nseList = nseCandidates
	
	//logrus.Infof("copy nseCandidates in nseList")
	copy(mg.nseList, nseCandidates) // Copy to avoid modifying original input later
	mg.nseNbr = n
	//logrus.Infof("mg.nseNbr  = %d ", mg.nseNbr)
	//logrus.Infof("end creation!")
	return nil
}



// Compute Hash values of all pods
func (mg *maglevSelector) ComputeHashValues() {

	//mg.permutation = nil
	mg.permutation =  make([][]uint64, mg.nseNbr)
	logrus.Infof("len(mg.nseList) %d vs mg.nseNbr %d vs mg.Lk_Size %d ", len(mg.nseList), mg.nseNbr, mg.Lk_Size)
	if len(mg.nseList) == 0 {
		return
	}
	
	
	//sort.Strings(mg.nseList)
	for i := 0; i < len(mg.nseList); i++ {

		bNseName := []byte(mg.nseList[i].GetName())
		logrus.Infof("bNseName %s for index %d vs Lk_size %d ", mg.nseList[i].GetName(), i, mg.Lk_Size)
	
		offset := siphash.Hash(0xdeadbabe, 0, bNseName) % mg.Lk_Size
		div := (mg.Lk_Size - 1)
		if mg.Lk_Size == 1 {
			div = 1
		}
		skip := (siphash.Hash(0xdeadbeef, 0, bNseName) % (div)) + 1
		logrus.Infof("offset %d skip %d mg.Lk_Size %d ", offset, skip, mg.Lk_Size)

		mg.permutation[i] =  make([]uint64, mg.Lk_Size)
		var j int 
		//uint64
		for j = 0; j < int(mg.Lk_Size); j++ {
			//iRow[j] = (offset + uint64(j)*skip) % mg.Lk_Size
			//logrus.Infof("len (permutation) = %d ", len(mg.permutation))
			logrus.Infof("len (permutation[i]) = %d ", len(mg.permutation[i]))
			mg.permutation[i][j] = (offset + uint64(j)*skip) % mg.Lk_Size
			logrus.Infof("permutation for i %d j %d = %d ", i, j, mg.permutation[i][j])
		}

	}

	// Add also network performance metrics??

}


// function Popoluate maglev hashing lookup table
func (mg *maglevSelector) PopulateMaglevHashing(requestConnection *connection.Connection, nseNbr int, ns *registry.NetworkService) {

	// Variable to save global pod selection per batch
	var LookupIdxPerRequestId = map[string]uint64{}
	
	// dynamic table Next

	var podID int
	var entry uint64
	Next := make([]uint64, mg.nseNbr)
	//lookupTable := make([]uint64, mg.Lk_Size)
	lookupTable := make(map[uint64]int, mg.Lk_Size)

	//Intialization of tables
	//logrus.Infof("Intialization of tables")
	for podID = 0; podID < int(mg.nseNbr); podID++ {
		Next[podID] = 0
	}
	for entry = 0; entry < mg.Lk_Size; entry++ {
		//initialVal := -1
		lookupTable[entry] = int(-1)
		//logrus.Infof("initial entry %d lookupTable[entry] %d ", entry, lookupTable[entry])
	}

	//logrus.Infof("compute permutations for mg.nseNbr %d ", mg.nseNbr)
	var n uint64
	for { // true
		for podID = 0; podID < int(mg.nseNbr); podID++ {
			//logrus.Infof("compute permutation for entry %d  and PODID %d Next[podID]  %d ", entry, podID, Next[podID])
			entry = mg.permutation[podID][Next[podID]]
			//logrus.Infof("entry %d lookupTable[entry] %d ", entry, lookupTable[entry])
			for lookupTable[entry] >= 0 {
				Next[podID] = Next[podID] + 1
				if Next[podID]>= mg.Lk_Size {
					break
				}
				logrus.Infof("Next[podID] %d entry %d lookupTable[entry] %d ", Next[podID], entry, lookupTable[entry])
				entry = mg.permutation[podID][Next[podID]]
			}
			// logrus.Infof("set entry %d in lookuptable to podID %d ", entry, podID)
			lookupTable[entry] = podID
		
			
			if requestConnection.GetId() == "" {
				logrus.Infof("requestConnection.Id cannot be empty: %v", requestConnection)
			}
			// else {
			requestName := strconv.FormatInt(int64(entry+1), 10)
			LookupIdxPerRequestId[requestName] = entry
			logrus.Infof("save requestName %s for entry %d ", requestName, entry)
			//}
			

			Next[podID] = Next[podID] + 1
			n++
			if n == mg.Lk_Size {
				mg.LookupTable = lookupTable
				logrus.Infof("lookupTable %v ",lookupTable)
				RequestIdPerNetworkService[ns.GetName()] = LookupIdxPerRequestId
				//logrus.Infof("set LookupTablePerNetworkService  %v", LookupTablePerNetworkService)
				LookupTablePerNetworkService[ns.GetName()] = mg.LookupTable
				//logrus.Infof("table fullfilled, return ")
				return
			}
		}
	}

}



func (mg *maglevSelector) SelectEndpoint(requestConnection *connection.Connection, ns *registry.NetworkService, networkServiceEndpoints []*registry.NetworkServiceEndpoint) *registry.NetworkServiceEndpoint {
	logrus.Infof("start SelectEndpoint for ns.name %s and requestConnection.getid %s ", ns.GetName(), requestConnection.GetId())

	//logrus.Infof("networkServiceEndpoints list %v ", networkServiceEndpoints)


	if mg == nil {
		logrus.Infof("return mg is nil ")
		return nil
	}
	if len(networkServiceEndpoints) == 0 {
		logrus.Infof("return networkServiceEndpoints is empty ")
		return nil
	}
	mg.Lock()
	defer mg.Unlock()

	
	// increment the number of requests per NS, which will be used as size of lookup table in maglev
	mg.maglev[ns.GetName()] =  mg.maglev[ns.GetName()] + 1

	logrus.Infof("number of requests in Lk of Maglev %d ", len(mg.maglev))

	// ####### create maglev slector #######
	mg.CreateMaglev(networkServiceEndpoints)

	var requestIdx uint64 = 0
	//var LookupIdxPerRequestId = map[string]int{}
	//var ok bool = true
	logrus.Infof("test if it is ok ")
	LookupIdxPerRequestId, ok := RequestIdPerNetworkService[ns.GetName()]

	if (ok) {

		logrus.Infof("Network service already existing ")
		
		// requestIdx, ok := RequestIdPerNetworkService[ns.GetName()][requestConnection.GetId()]

		IdxPerRequestId, ok2 := LookupIdxPerRequestId[requestConnection.GetId()]
		ConvertReqId, err := strconv.Atoi(requestConnection.GetId())
		if err != nil {
			// handle error
		}
		logrus.Infof(" ConvertReqId %d",ConvertReqId)
		if (ok2) && RequestIdPerNetworkService[ns.GetName()] != nil && (ConvertReqId == int(IdxPerRequestId)) {
			// get request Idx
			requestIdx = IdxPerRequestId
			//LookupIdxPerRequestId[requestConnection.GetId()]
			logrus.Infof("requestIdx %d requestId %s ", requestIdx, requestConnection.GetId())
			mg.LookupTable = LookupTablePerNetworkService[ns.GetName()]

		} else {
			// Compute hash values in permutation[][]
			logrus.Infof("Compute for request consistent hashing with Maglev ")
			mg.ComputeHashValues()
			// Compute Maglev consistent hashing decision
			mg.PopulateMaglevHashing(requestConnection, len(networkServiceEndpoints), ns)
			requestIdx = LookupIdxPerRequestId[requestConnection.GetId()]
			logrus.Infof("requestIdx %d requestId %s ", requestIdx, requestConnection.GetId())
		}
	} else { // add the network service to map

		// Compute hash values in permutation[][]
		logrus.Infof("Compute consistent hashing with Maglev ")

		mg.ComputeHashValues()
		// Compute Maglev consistent hashing decision
		
		mg.PopulateMaglevHashing(requestConnection, len(networkServiceEndpoints), ns)

		//logrus.Infof("get requestIdx ")
		requestIdx = RequestIdPerNetworkService[ns.GetName()][requestConnection.GetId()]
		logrus.Infof("get requestIdx %d requestId %s RequestIdPerNetworkService %v ", requestIdx, requestConnection.GetId(), RequestIdPerNetworkService)

	}

	// Get selected NSE pod index
	logrus.Infof("get index for requestIdx %d requestName %s ", requestIdx, requestConnection.GetId())
	idx := LookupTablePerNetworkService[ns.GetName()][requestIdx]
	
	logrus.Infof("get endpoint for idx %d vs len(nsEndpoints) %d ", idx, len(networkServiceEndpoints))

	endpoint := networkServiceEndpoints[idx]

	if endpoint == nil {
		logrus.Infof("selected endpoint nil %v for idx %d ", endpoint, idx)
		return nil
	}
	//mg.maglev[ns.GetName()] = mg.maglev[ns.GetName()] + 1
	logrus.Infof("maglev selected %v with index %d ", endpoint, idx)
	
	logrus.Infof("maglev selected endpoint name %s", endpoint.GetName())
	return endpoint
}
