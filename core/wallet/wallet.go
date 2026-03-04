package wallet

import (
	"sync"

	ipfsnode "github.com/ipfs/go-ipfs-api"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rubixchain/rubixgoplatform/core/model"
	"github.com/rubixchain/rubixgoplatform/core/storage"
	"github.com/rubixchain/rubixgoplatform/wrapper/logger"
	"github.com/syndtr/goleveldb/leveldb"
)

const (
	TokenStorage                   string = "TokensTable"
	NFTTokenStorage                string = "NFTTokensTable"
	CreditStorage                  string = "CreditsTable"
	DIDStorage                     string = "DIDTable"
	DIDPeerStorage                 string = "DIDPeerTable"
	TransactionStorage             string = "TransactionHistory"
	TokensArrayStorage             string = "TokensTransferred"
	TokenProvider                  string = "TokenProviderTable"
	TokenChainStorage              string = "tokenchainstorage"
	NFTChainStorage                string = "nftchainstorage"
	SmartContractTokenChainStorage string = "smartcontractokenchainstorage"
	SmartContractStorage           string = "smartcontract"
	CallBackUrlStorage             string = "callbackurl"
	TokenStateHash                 string = "TokenStateHashTable"
	UnpledgeQueueTable             string = "unpledgequeue"
	UnpledgeSequence               string = "UnpledgeSequence"
	FTTokenStorage                 string = "FTTokenTable"
	FTChainStorage                 string = "FTchainstorage"
	FTStorage                      string = "FTTable"
	FTTransactionTokenStorage      string = "FTTransactionTokens"
	FailedFTDownloadStorage        string = "FailedFTDownloads"
	FullNodeStorage                string = "Fullnodestorage"
	FullNodeRBTTable               string = "FullnodeRBTtable"
	FullNodeFTTable                string = "FullnodeFTtable"
	FullNodeNFTTable               string = "FullnodeNFTtable"
	FullNodeSmartContractTable     string = "FullnodeSCtable"
	FullNodeTxnHistoryTable        string = "FullnodeTxnHistoryTable"
	FailedTxnsTable                string = "FailedTxns"
	FullNodeFailedToSyncTokens     string = "FullnodeFailedTokensTable"
	FullNodeRBTContentTable        string = "rbt_content_table"
	FullNodeFTContentTable         string = "ft_content_table"
	FullNodeNFTContentTable        string = "nft_content_table"
	FullNodeSCContentTable         string = "sc_content_table"
	FullnodeDoubleSpentTokensTable string = "DoubleSpentTokensTable"
	LocalTestTokenInfo             string = "LocalTestTokenInfo"
)

type WalletConfig struct {
	StorageType   int    `json:"stroage_type"`
	DBName        string `json:"db_name"`
	DBAddress     string `json:"db_address"`
	DBPort        string `json:"db_port"`
	DBType        string `json:"db_type"`
	DBUserName    string `json:"db_user_name"`
	DBPassword    string `json:"db_password"`
	// Deprecated: TokenChainDir was used for LevelDB storage, now unused with PostgreSQL.
	TokenChainDir string `json:"token_chain_dir"`
}

type ChainDB struct {
	*leveldb.DB
	l sync.Mutex
}

type Wallet struct {
	ipfs                           *ipfsnode.Shell
	ipfsOps                        IPFSOperations
	s                              storage.Storage
	fullNodeSQLDB                  storage.Storage
	fullNodePSQLTokensDB           storage.Storage
	l                              sync.Mutex
	dtl                            sync.Mutex
	log                            logger.Logger
	wl                             sync.Mutex
	tcs                            *ChainDB
	ntcs                           *ChainDB
	smartContractTokenChainStorage *ChainDB
	FTChainStorage                 *ChainDB
	asyncProviderMgr               *AsyncProviderDetailsManager
	fullNodeStorage                *ChainDB
	IsFullNode                     bool
	pool                           *pgxpool.Pool
}

// GetStorage returns the storage interface
func (w *Wallet) GetStorage() storage.Storage {
	return w.s
}

// GetIpfsOps returns the IPFS operations interface
func (w *Wallet) GetIpfsOps() IPFSOperations {
	return w.ipfsOps
}

// GetPool returns the pgxpool.Pool for direct PostgreSQL access.
func (w *Wallet) GetPool() *pgxpool.Pool {
	return w.pool
}

func InitWallet(s storage.Storage, fullNodeSQLDB storage.Storage, fullNodePSQLTokensDB storage.Storage, dir string, log logger.Logger, fullNode bool) (*Wallet, error) {
	var err error
	w := &Wallet{
		log:                  log.Named("wallet"),
		s:                    s,
		fullNodeSQLDB:        fullNodeSQLDB,
		fullNodePSQLTokensDB: fullNodePSQLTokensDB,
		IsFullNode:           fullNode,
	}
	// ChainDB fields are kept for compatibility with token_chain.go but are not
	// backed by LevelDB. LevelDB storage is deprecated; use PostgreSQL instead.
	w.tcs = &ChainDB{}
	w.ntcs = &ChainDB{}
	w.smartContractTokenChainStorage = &ChainDB{}
	w.FTChainStorage = &ChainDB{}
	w.fullNodeStorage = &ChainDB{}

	err = w.s.Init(DIDStorage, &DID{}, true)
	if err != nil {
		w.log.Error("Failed to initialize DID storage", "err", err)
		return nil, err
	}
	err = w.s.Init(TokenStorage, &Token{}, true)
	if err != nil {
		w.log.Error("Failed to initialize whole token storage", "err", err)
		return nil, err
	}
	err = w.s.Init(NFTTokenStorage, &NFT{}, true)
	if err != nil {
		w.log.Error("Failed to initialize data token storage", "err", err)
		return nil, err
	}
	err = w.s.Init(CreditStorage, &Credit{}, true)
	if err != nil {
		w.log.Error("Failed to initialize credit storage", "err", err)
		return nil, err
	}
	err = w.s.Init(DIDPeerStorage, &DIDPeerMap{}, true)
	if err != nil {
		w.log.Error("Failed to initialize DID Peer storage", "err", err)
		return nil, err
	}
	err = w.s.Init(TransactionStorage, &model.TransactionDetails{}, true)
	if err != nil {
		w.log.Error("Failed to initialize Transaction storage", "err", err)
		return nil, err
	}
	err = w.s.Init(TokenProvider, &model.TokenProviderMap{}, true)
	if err != nil {
		w.log.Error("Failed to initialize Token Provider Table", "err", err)
		return nil, err
	}
	err = w.s.Init(SmartContractStorage, &SmartContract{}, true)
	if err != nil {
		w.log.Error("Failed to initialize Smart Contract storage", "err", err)
		return nil, err
	}
	err = w.s.Init(UnpledgeSequence, &UnpledgeSequenceInfo{}, true)
	if err != nil {
		w.log.Error("failed to init UnpledgeSequence table", "err", err)
		return nil, err
	}
	err = w.s.Init(FTTokenStorage, FTToken{}, true)
	if err != nil {
		w.log.Error("Failed to initialize FT Token storage", "err", err)
		return nil, err
	}
	err = w.s.Init(FTStorage, &FT{}, true)
	if err != nil {
		w.log.Error("Failed to initialize FT storage", "err", err)
		return nil, err
	}
	err = w.s.Init(FTTransactionTokenStorage, &model.FTTransactionToken{}, true)
	if err != nil {
		w.log.Error("Failed to initialize FT transaction token storage", "err", err)
		return nil, err
	}
	err = w.s.Init(FTTransactionHistoryStorage, &model.FTTransactionHistory{}, true)
	if err != nil {
		w.log.Error("Failed to initialize FT transaction history storage", "err", err)
		return nil, err
	}
	// Initialize token recovery tracking table
	err = w.s.Init("TokenRecovery", &model.TokenRecovery{}, true)
	if err != nil {
		w.log.Error("Failed to initialize token recovery storage", "err", err)
		return nil, err
	}

	err = w.s.Init(LocalTestTokenInfo, &model.LocalTestTokenInfo{}, true)
	if err != nil {
		w.log.Error("Failed to initialize local test token tracking storage", "err", err)
		return nil, err
	}

	// Initialize normalized SQL tables introduced in P1-T01/T02.
	err = w.s.Init("transactions", &TransactionRecord{}, true)
	if err != nil {
		w.log.Error("Failed to initialize transactions storage", "err", err)
		return nil, err
	}
	err = w.s.Init("tokenchain", &TokenChainEntry{}, true)
	if err != nil {
		w.log.Error("Failed to initialize tokenchain storage", "err", err)
		return nil, err
	}
	err = w.s.Init("requests", &RequestRecord{}, true)
	if err != nil {
		w.log.Error("Failed to initialize requests storage", "err", err)
		return nil, err
	}

	// If the underlying storage is a SQL DB, apply DDL for indexes and FK constraints
	// that GORM's AutoMigrate does not express via struct tags alone.
	if sdb, ok := w.s.(*storage.StorageDB); ok {
		// Composite index for fast "latest entry per token" queries.
		if err = sdb.ExecSQL(`CREATE INDEX IF NOT EXISTS idx_tokenchain_latest ON tokenchain(token_id, position DESC)`); err != nil {
			w.log.Error("Failed to create idx_tokenchain_latest index", "err", err)
			return nil, err
		}
		// FK: tokenchain.transaction_id -> transactions.id (using a DO block for idempotency on PostgreSQL).
		if err = sdb.ExecSQL(`DO $$ BEGIN
  ALTER TABLE tokenchain ADD CONSTRAINT fk_tc_tx FOREIGN KEY(transaction_id) REFERENCES transactions(id) ON DELETE RESTRICT;
EXCEPTION WHEN duplicate_object THEN NULL;
END $$`); err != nil {
			w.log.Error("Failed to add fk_tc_tx constraint", "err", err)
			return nil, err
		}
		// FK: tokens.transaction_id -> transactions.id (deferred to allow within-transaction ordering).
		if err = sdb.ExecSQL(`DO $$ BEGIN
  ALTER TABLE tokens ADD CONSTRAINT fk_tok_latest_tx FOREIGN KEY(transaction_id) REFERENCES transactions(id) DEFERRABLE INITIALLY DEFERRED;
EXCEPTION WHEN duplicate_object THEN NULL;
END $$`); err != nil {
			w.log.Error("Failed to add fk_tok_latest_tx constraint", "err", err)
			return nil, err
		}
	}

	err = w.s.Init(CallBackUrlStorage, &CallBackUrl{}, true)
	if err != nil {
		w.log.Error("Failed to initialize Smart Contract Callback Url storage", "err", err)
		return nil, err
	}

	err = w.s.Init(TokenStateHash, &TokenStateDetails{}, true)
	if err != nil {
		w.log.Error("Failed to initialize TokenStateHash", "err", err)
		return nil, err
	}

	// Initialize async provider details manager with 2 workers
	w.asyncProviderMgr = NewAsyncProviderDetailsManager(w, 2)

	// Initialize async provider details manager with 2 workers
	w.asyncProviderMgr = NewAsyncProviderDetailsManager(w, 2)

	// DB for fullnodes to store all token-chains (LevelDB removed; fullNodeStorage is no-op).
	if w.IsFullNode {
		err = w.fullNodeSQLDB.Init(FullNodeRBTTable, &SyncedRBT{}, true)
		if err != nil {
			w.log.Error("Failed to initialize RBT token storage", "err", err)
			return nil, err
		}

		err = w.fullNodeSQLDB.Init(FullNodeFTTable, &SyncedFT{}, true)
		if err != nil {
			w.log.Error("Failed to initialize FT token storage", "err", err)
			return nil, err
		}

		err = w.fullNodeSQLDB.Init(FullNodeNFTTable, &SyncedNFT{}, true)
		if err != nil {
			w.log.Error("Failed to initialize NFT token storage", "err", err)
			return nil, err
		}

		err = w.fullNodeSQLDB.Init(FullNodeSmartContractTable, &SyncedSmartContract{}, true)
		if err != nil {
			w.log.Error("Failed to initialize fullnode smart contract token storage", "err", err)
			return nil, err
		}

		err = w.fullNodeSQLDB.Init(FailedTxnsTable, &model.FailedTransaction{}, true)
		if err != nil {
			w.log.Error("Failed to initialize fullnode failed transaction storage", "err", err)
			return nil, err
		}
		err = w.fullNodeSQLDB.Init(FullNodeFailedToSyncTokens, &model.FailedToSyncTokenDetailsInfo{}, true)
		if err != nil {
			w.log.Error("failed to initialize FullNodeFailedToSyncTokens storage", "error", err)
			return nil, err
		}

		err = w.fullNodeSQLDB.Init(FullnodeDoubleSpentTokensTable, &model.DoubleSpentTokenInfo{}, true)
		if err != nil {
			w.log.Error("failed to initialize FullnodeDoubleSpentTokensTable storage", "error", err)
			return nil, err
		}
		err = w.fullNodeSQLDB.Init(FullNodeTxnHistoryTable, &model.FullNodeTxnHistoryInfo{}, true)
		if err != nil {
			w.log.Error("failed to initialize FullNodeTxnHistoryTable storage", "error", err)
		}

		err = w.fullNodePSQLTokensDB.Init(FullNodeRBTContentTable, &RBTContent{}, true)
		if err != nil {
			w.log.Error("Failed to initialize postgres RBT token storage", "err", err)
			return nil, err
		}

		err = w.fullNodePSQLTokensDB.Init(FullNodeFTContentTable, &FTContent{}, true)
		if err != nil {
			w.log.Error("Failed to initialize postgres FT token storage", "err", err)
			return nil, err
		}

		err = w.fullNodePSQLTokensDB.Init(FullNodeNFTContentTable, &NFTContent{}, true)
		if err != nil {
			w.log.Error("Failed to initialize NFT token storage", "err", err)
			return nil, err
		}

		err = w.fullNodePSQLTokensDB.Init(FullNodeSCContentTable, &SmartContractContent{}, true)
		if err != nil {
			w.log.Error("Failed to initialize fullnode smart contract token storage", "err", err)
			return nil, err
		}
	}

	return w, nil
}

func (w *Wallet) SetupWallet(ipfs *ipfsnode.Shell) {
	w.ipfs = ipfs
	// Default to direct IPFS operations if no health-managed operations are set
	if w.ipfsOps == nil {
		w.ipfsOps = NewDirectIPFSOperations(ipfs)
	}
}

// SetIPFSOperations sets the IPFS operations interface (for health-managed operations)
func (w *Wallet) SetIPFSOperations(ops IPFSOperations) {
	w.ipfsOps = ops
}

// Re-export StorageType for convenience
// StorageType is used for batch writes (Key, Value)
type StorageType = storage.StorageType

// S returns the storage interface (for batch writes)
func (w *Wallet) S() storage.Storage {
	return w.s
}
