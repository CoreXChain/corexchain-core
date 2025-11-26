// Copyright 2015 CoreX Team
// This file is part of the corex-chain library.
//
// The corex-chain library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The corex-chain library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the corex-chain library. If not, see <http://www.gnu.org/licenses/>.

// Package web3ext contains corex specific web3.js extensions.
package web3ext

var Modules = map[string]string{
	"account":       AccountJs,
	"admin":         AdminJs,
	"asset":         AssetJs,
	"clique":        CliqueJs,
	"communication": CommunicationJs,
	"control":       ControlJs,
	"corexhash":     CoreXHashJs,
	"debug":         DebugJs,
	"corex":         CoreJs,
	"identity":      IdentityJs,
	"net":           NetJs,
	"pov":           PoVJs,
	"production":    ProductionJs,
	"rpc":           RpcJs,
	"trust":         TrustJs,
	"valueflow":     ValueFlowJs,
	"les":           LESJs,
	"vflux":         VfluxJs,
	"dev":           DevJs,
}

const CliqueJs = `
web3._extend({
	property: 'clique',
	methods: [
		new web3._extend.Method({
			name: 'getSnapshot',
			call: 'clique_getSnapshot',
			params: 1,
			inputFormatter: [web3._extend.formatters.inputBlockNumberFormatter]
		}),
		new web3._extend.Method({
			name: 'getSnapshotAtHash',
			call: 'clique_getSnapshotAtHash',
			params: 1
		}),
		new web3._extend.Method({
			name: 'getSigners',
			call: 'clique_getSigners',
			params: 1,
			inputFormatter: [web3._extend.formatters.inputBlockNumberFormatter]
		}),
		new web3._extend.Method({
			name: 'getSignersAtHash',
			call: 'clique_getSignersAtHash',
			params: 1
		}),
		new web3._extend.Method({
			name: 'propose',
			call: 'clique_propose',
			params: 2
		}),
		new web3._extend.Method({
			name: 'discard',
			call: 'clique_discard',
			params: 1
		}),
		new web3._extend.Method({
			name: 'status',
			call: 'clique_status',
			params: 0
		}),
		new web3._extend.Method({
			name: 'getSigner',
			call: 'clique_getSigner',
			params: 1,
			inputFormatter: [null]
		}),
	],
	properties: [
		new web3._extend.Property({
			name: 'proposals',
			getter: 'clique_proposals'
		}),
	]
});
`

const CoreXHashJs = `
web3._extend({
	property: 'corexhash',
	methods: [
		new web3._extend.Method({
			name: 'getWork',
			call: 'corexhash_getWork',
			params: 0
		}),
		new web3._extend.Method({
			name: 'getHashrate',
			call: 'corexhash_getHashrate',
			params: 0
		}),
		new web3._extend.Method({
			name: 'submitWork',
			call: 'corexhash_submitWork',
			params: 3,
		}),
		new web3._extend.Method({
			name: 'submitHashrate',
			call: 'corexhash_submitHashrate',
			params: 2,
		}),
	]
});
`

const AdminJs = `
web3._extend({
	property: 'admin',
	methods: [
		new web3._extend.Method({
			name: 'addPeer',
			call: 'admin_addPeer',
			params: 1
		}),
		new web3._extend.Method({
			name: 'removePeer',
			call: 'admin_removePeer',
			params: 1
		}),
		new web3._extend.Method({
			name: 'addTrustedPeer',
			call: 'admin_addTrustedPeer',
			params: 1
		}),
		new web3._extend.Method({
			name: 'removeTrustedPeer',
			call: 'admin_removeTrustedPeer',
			params: 1
		}),
		new web3._extend.Method({
			name: 'exportChain',
			call: 'admin_exportChain',
			params: 3,
			inputFormatter: [null, null, null]
		}),
		new web3._extend.Method({
			name: 'importChain',
			call: 'admin_importChain',
			params: 1
		}),
		new web3._extend.Method({
			name: 'sleepBlocks',
			call: 'admin_sleepBlocks',
			params: 2
		}),
		new web3._extend.Method({
			name: 'startHTTP',
			call: 'admin_startHTTP',
			params: 5,
			inputFormatter: [null, null, null, null, null]
		}),
		new web3._extend.Method({
			name: 'stopHTTP',
			call: 'admin_stopHTTP'
		}),
		// This method is deprecated.
		new web3._extend.Method({
			name: 'startRPC',
			call: 'admin_startRPC',
			params: 5,
			inputFormatter: [null, null, null, null, null]
		}),
		// This method is deprecated.
		new web3._extend.Method({
			name: 'stopRPC',
			call: 'admin_stopRPC'
		}),
		new web3._extend.Method({
			name: 'startWS',
			call: 'admin_startWS',
			params: 4,
			inputFormatter: [null, null, null, null]
		}),
		new web3._extend.Method({
			name: 'stopWS',
			call: 'admin_stopWS'
		}),
	],
	properties: [
		new web3._extend.Property({
			name: 'nodeInfo',
			getter: 'admin_nodeInfo'
		}),
		new web3._extend.Property({
			name: 'peers',
			getter: 'admin_peers'
		}),
		new web3._extend.Property({
			name: 'datadir',
			getter: 'admin_datadir'
		}),
	]
});
`

const ControlJs = `
web3._extend({
	property: 'control',
	methods: [
		new web3._extend.Method({
			name: 'joinPeer',
			call: 'control_joinPeer',
			params: 1
		}),
		new web3._extend.Method({
			name: 'dropPeer',
			call: 'control_dropPeer',
			params: 1
		}),
		new web3._extend.Method({
			name: 'joinTrustedPeer',
			call: 'control_joinTrustedPeer',
			params: 1
		}),
		new web3._extend.Method({
			name: 'dropTrustedPeer',
			call: 'control_dropTrustedPeer',
			params: 1
		}),
		new web3._extend.Method({
			name: 'exportLedger',
			call: 'control_exportLedger',
			params: 3,
			inputFormatter: [null, null, null]
		}),
		new web3._extend.Method({
			name: 'importLedger',
			call: 'control_importLedger',
			params: 1
		}),
		new web3._extend.Method({
			name: 'startHTTP',
			call: 'control_startHTTP',
			params: 5,
			inputFormatter: [null, null, null, null, null]
		}),
		new web3._extend.Method({
			name: 'stopHTTP',
			call: 'control_stopHTTP'
		}),
		new web3._extend.Method({
			name: 'startWS',
			call: 'control_startWS',
			params: 4,
			inputFormatter: [null, null, null, null]
		}),
		new web3._extend.Method({
			name: 'stopWS',
			call: 'control_stopWS'
		}),
	],
	properties: [
		new web3._extend.Property({
			name: 'nodeProfile',
			getter: 'control_nodeProfile'
		}),
		new web3._extend.Property({
			name: 'peers',
			getter: 'control_peers'
		}),
		new web3._extend.Property({
			name: 'workspace',
			getter: 'control_workspace'
		}),
	]
});
`

const DebugJs = `
web3._extend({
	property: 'debug',
	methods: [
		new web3._extend.Method({
			name: 'accountRange',
			call: 'debug_accountRange',
			params: 6,
			inputFormatter: [web3._extend.formatters.inputDefaultBlockNumberFormatter, null, null, null, null, null],
		}),
		new web3._extend.Method({
			name: 'printBlock',
			call: 'debug_printBlock',
			params: 1,
			outputFormatter: console.log
		}),
		new web3._extend.Method({
			name: 'getRawHeader',
			call: 'debug_getRawHeader',
			params: 1
		}),
		new web3._extend.Method({
			name: 'getRawBlock',
			call: 'debug_getRawBlock',
			params: 1
		}),
		new web3._extend.Method({
			name: 'getRawReceipts',
			call: 'debug_getRawReceipts',
			params: 1
		}),
		new web3._extend.Method({
			name: 'getRawTransaction',
			call: 'debug_getRawTransaction',
			params: 1
		}),
		new web3._extend.Method({
			name: 'setHead',
			call: 'debug_setHead',
			params: 1
		}),
		new web3._extend.Method({
			name: 'seedHash',
			call: 'debug_seedHash',
			params: 1
		}),
		new web3._extend.Method({
			name: 'dumpBlock',
			call: 'debug_dumpBlock',
			params: 1,
			inputFormatter: [web3._extend.formatters.inputBlockNumberFormatter]
		}),
		new web3._extend.Method({
			name: 'chaindbProperty',
			call: 'debug_chaindbProperty',
			params: 1,
			outputFormatter: console.log
		}),
		new web3._extend.Method({
			name: 'chaindbCompact',
			call: 'debug_chaindbCompact',
		}),
		new web3._extend.Method({
			name: 'verbosity',
			call: 'debug_verbosity',
			params: 1
		}),
		new web3._extend.Method({
			name: 'vmodule',
			call: 'debug_vmodule',
			params: 1
		}),
		new web3._extend.Method({
			name: 'backtraceAt',
			call: 'debug_backtraceAt',
			params: 1,
		}),
		new web3._extend.Method({
			name: 'stacks',
			call: 'debug_stacks',
			params: 1,
			inputFormatter: [null],
			outputFormatter: console.log
		}),
		new web3._extend.Method({
			name: 'freeOSMemory',
			call: 'debug_freeOSMemory',
			params: 0,
		}),
		new web3._extend.Method({
			name: 'setGCPercent',
			call: 'debug_setGCPercent',
			params: 1,
		}),
		new web3._extend.Method({
			name: 'memStats',
			call: 'debug_memStats',
			params: 0,
		}),
		new web3._extend.Method({
			name: 'gcStats',
			call: 'debug_gcStats',
			params: 0,
		}),
		new web3._extend.Method({
			name: 'cpuProfile',
			call: 'debug_cpuProfile',
			params: 2
		}),
		new web3._extend.Method({
			name: 'startCPUProfile',
			call: 'debug_startCPUProfile',
			params: 1
		}),
		new web3._extend.Method({
			name: 'stopCPUProfile',
			call: 'debug_stopCPUProfile',
			params: 0
		}),
		new web3._extend.Method({
			name: 'goTrace',
			call: 'debug_goTrace',
			params: 2
		}),
		new web3._extend.Method({
			name: 'startGoTrace',
			call: 'debug_startGoTrace',
			params: 1
		}),
		new web3._extend.Method({
			name: 'stopGoTrace',
			call: 'debug_stopGoTrace',
			params: 0
		}),
		new web3._extend.Method({
			name: 'blockProfile',
			call: 'debug_blockProfile',
			params: 2
		}),
		new web3._extend.Method({
			name: 'setBlockProfileRate',
			call: 'debug_setBlockProfileRate',
			params: 1
		}),
		new web3._extend.Method({
			name: 'writeBlockProfile',
			call: 'debug_writeBlockProfile',
			params: 1
		}),
		new web3._extend.Method({
			name: 'mutexProfile',
			call: 'debug_mutexProfile',
			params: 2
		}),
		new web3._extend.Method({
			name: 'setMutexProfileFraction',
			call: 'debug_setMutexProfileFraction',
			params: 1
		}),
		new web3._extend.Method({
			name: 'writeMutexProfile',
			call: 'debug_writeMutexProfile',
			params: 1
		}),
		new web3._extend.Method({
			name: 'writeMemProfile',
			call: 'debug_writeMemProfile',
			params: 1
		}),
		new web3._extend.Method({
			name: 'traceBlock',
			call: 'debug_traceBlock',
			params: 2,
			inputFormatter: [null, null]
		}),
		new web3._extend.Method({
			name: 'traceBlockFromFile',
			call: 'debug_traceBlockFromFile',
			params: 2,
			inputFormatter: [null, null]
		}),
		new web3._extend.Method({
			name: 'traceBadBlock',
			call: 'debug_traceBadBlock',
			params: 1,
			inputFormatter: [null]
		}),
		new web3._extend.Method({
			name: 'standardTraceBadBlockToFile',
			call: 'debug_standardTraceBadBlockToFile',
			params: 2,
			inputFormatter: [null, null]
		}),
		new web3._extend.Method({
			name: 'intermediateRoots',
			call: 'debug_intermediateRoots',
			params: 2,
			inputFormatter: [null, null]
		}),
		new web3._extend.Method({
			name: 'standardTraceBlockToFile',
			call: 'debug_standardTraceBlockToFile',
			params: 2,
			inputFormatter: [null, null]
		}),
		new web3._extend.Method({
			name: 'traceBlockByNumber',
			call: 'debug_traceBlockByNumber',
			params: 2,
			inputFormatter: [web3._extend.formatters.inputBlockNumberFormatter, null]
		}),
		new web3._extend.Method({
			name: 'traceBlockByHash',
			call: 'debug_traceBlockByHash',
			params: 2,
			inputFormatter: [null, null]
		}),
		new web3._extend.Method({
			name: 'traceTransaction',
			call: 'debug_traceTransaction',
			params: 2,
			inputFormatter: [null, null]
		}),
		new web3._extend.Method({
			name: 'traceCall',
			call: 'debug_traceCall',
			params: 3,
			inputFormatter: [null, null, null]
		}),
		new web3._extend.Method({
			name: 'preimage',
			call: 'debug_preimage',
			params: 1,
			inputFormatter: [null]
		}),
		new web3._extend.Method({
			name: 'getBadBlocks',
			call: 'debug_getBadBlocks',
			params: 0,
		}),
		new web3._extend.Method({
			name: 'storageRangeAt',
			call: 'debug_storageRangeAt',
			params: 5,
		}),
		new web3._extend.Method({
			name: 'getModifiedAccountsByNumber',
			call: 'debug_getModifiedAccountsByNumber',
			params: 2,
			inputFormatter: [null, null],
		}),
		new web3._extend.Method({
			name: 'getModifiedAccountsByHash',
			call: 'debug_getModifiedAccountsByHash',
			params: 2,
			inputFormatter:[null, null],
		}),
		new web3._extend.Method({
			name: 'freezeClient',
			call: 'debug_freezeClient',
			params: 1,
		}),
		new web3._extend.Method({
			name: 'getAccessibleState',
			call: 'debug_getAccessibleState',
			params: 2,
			inputFormatter:[web3._extend.formatters.inputBlockNumberFormatter, web3._extend.formatters.inputBlockNumberFormatter],
		}),
		new web3._extend.Method({
			name: 'dbGet',
			call: 'debug_dbGet',
			params: 1
		}),
		new web3._extend.Method({
			name: 'dbAncient',
			call: 'debug_dbAncient',
			params: 2
		}),
		new web3._extend.Method({
			name: 'dbAncients',
			call: 'debug_dbAncients',
			params: 0
		}),
		new web3._extend.Method({
			name: 'setTrieFlushInterval',
			call: 'debug_setTrieFlushInterval',
			params: 1
		}),
		new web3._extend.Method({
			name: 'getTrieFlushInterval',
			call: 'debug_getTrieFlushInterval',
			params: 0
		}),
	],
	properties: []
});
`

const CoreJs = `
web3._extend({
	property: 'corex',
	methods: [
		new web3._extend.Method({
			name: 'chainId',
			call: 'corex_chainId',
			params: 0
		}),
		new web3._extend.Method({
			name: 'sign',
			call: 'corex_sign',
			params: 2,
			inputFormatter: [web3._extend.formatters.inputAddressFormatter, null]
		}),
		new web3._extend.Method({
			name: 'resend',
			call: 'corex_resend',
			params: 3,
			inputFormatter: [web3._extend.formatters.inputTransactionFormatter, web3._extend.utils.fromDecimal, web3._extend.utils.fromDecimal]
		}),
		new web3._extend.Method({
			name: 'signTransaction',
			call: 'corex_signTransaction',
			params: 1,
			inputFormatter: [web3._extend.formatters.inputTransactionFormatter]
		}),
		new web3._extend.Method({
			name: 'estimateGas',
			call: 'corex_estimateGas',
			params: 3,
			inputFormatter: [web3._extend.formatters.inputCallFormatter, web3._extend.formatters.inputBlockNumberFormatter, null],
			outputFormatter: web3._extend.utils.toDecimal
		}),
		new web3._extend.Method({
			name: 'submitTransaction',
			call: 'corex_submitTransaction',
			params: 1,
			inputFormatter: [web3._extend.formatters.inputTransactionFormatter]
		}),
		new web3._extend.Method({
			name: 'fillTransaction',
			call: 'corex_fillTransaction',
			params: 1,
			inputFormatter: [web3._extend.formatters.inputTransactionFormatter]
		}),
		new web3._extend.Method({
			name: 'getHeaderByNumber',
			call: 'corex_getHeaderByNumber',
			params: 1,
			inputFormatter: [web3._extend.formatters.inputBlockNumberFormatter]
		}),
		new web3._extend.Method({
			name: 'getHeaderByHash',
			call: 'corex_getHeaderByHash',
			params: 1
		}),
		new web3._extend.Method({
			name: 'getBlockByNumber',
			call: 'corex_getBlockByNumber',
			params: 2,
			inputFormatter: [web3._extend.formatters.inputBlockNumberFormatter, function (val) { return !!val; }]
		}),
		new web3._extend.Method({
			name: 'getBlockByHash',
			call: 'corex_getBlockByHash',
			params: 2,
			inputFormatter: [null, function (val) { return !!val; }]
		}),
		new web3._extend.Method({
			name: 'getRawTransaction',
			call: 'corex_getRawTransactionByHash',
			params: 1
		}),
		new web3._extend.Method({
			name: 'getRawTransactionFromBlock',
			call: function(args) {
				return (web3._extend.utils.isString(args[0]) && args[0].indexOf('0x') === 0) ? 'corex_getRawTransactionByBlockHashAndIndex' : 'corex_getRawTransactionByBlockNumberAndIndex';
			},
			params: 2,
			inputFormatter: [web3._extend.formatters.inputBlockNumberFormatter, web3._extend.utils.toHex]
		}),
		new web3._extend.Method({
			name: 'getProof',
			call: 'corex_getProof',
			params: 3,
			inputFormatter: [web3._extend.formatters.inputAddressFormatter, null, web3._extend.formatters.inputBlockNumberFormatter]
		}),
		new web3._extend.Method({
			name: 'createAccessProfile',
			call: 'corex_createAccessProfile',
			params: 2,
			inputFormatter: [null, web3._extend.formatters.inputBlockNumberFormatter],
		}),
		new web3._extend.Method({
			name: 'feeHistory',
			call: 'corex_feeHistory',
			params: 3,
			inputFormatter: [null, web3._extend.formatters.inputBlockNumberFormatter, null]
		}),
		new web3._extend.Method({
			name: 'getLogs',
			call: 'corex_getLogs',
			params: 1,
		}),
		new web3._extend.Method({
			name: 'call',
			call: 'corex_call',
			params: 4,
			inputFormatter: [web3._extend.formatters.inputCallFormatter, web3._extend.formatters.inputDefaultBlockNumberFormatter, null, null],
		}),
		new web3._extend.Method({
			name: 'getBlockReceipts',
			call: 'corex_getBlockReceipts',
			params: 1,
		}),
	],
	properties: [
		new web3._extend.Property({
			name: 'pendingTransactions',
			getter: 'corex_pendingTransactions',
			outputFormatter: function(txs) {
				var formatted = [];
				for (var i = 0; i < txs.length; i++) {
					formatted.push(web3._extend.formatters.outputTransactionFormatter(txs[i]));
					formatted[i].blockHash = null;
				}
				return formatted;
			}
		}),
		new web3._extend.Property({
			name: 'maxPriorityFeePerGas',
			getter: 'corex_maxPriorityFeePerGas',
			outputFormatter: web3._extend.utils.toBigNumber
		}),
	]
});
`

const ProductionJs = `
web3._extend({
	property: 'production',
	methods: [
		new web3._extend.Method({
			name: 'start',
			call: 'production_start',
		}),
		new web3._extend.Method({
			name: 'stop',
			call: 'production_stop'
		}),
		new web3._extend.Method({
			name: 'setRewardBase',
			call: 'production_setRewardBase',
			params: 1,
			inputFormatter: [web3._extend.formatters.inputAddressFormatter]
		}),
		new web3._extend.Method({
			name: 'setMetadata',
			call: 'production_setMetadata',
			params: 1
		}),
		new web3._extend.Method({
			name: 'setFeeFloor',
			call: 'production_setFeeFloor',
			params: 1,
			inputFormatter: [web3._extend.utils.fromDecimal]
		}),
		new web3._extend.Method({
			name: 'setBlockGasLimit',
			call: 'production_setBlockGasLimit',
			params: 1,
			inputFormatter: [web3._extend.utils.fromDecimal]
		}),
		new web3._extend.Method({
			name: 'setRecommitInterval',
			call: 'production_setRecommitInterval',
			params: 1,
		}),
		new web3._extend.Method({
			name: 'getProductionRate',
			call: 'production_getProductionRate'
		}),
	],
	properties: []
});
`

const NetJs = `
web3._extend({
	property: 'net',
	methods: [],
	properties: [
		new web3._extend.Property({
			name: 'version',
			getter: 'net_version'
		}),
	]
});
`

const IdentityJs = `
web3._extend({
	property: 'identity',
	methods: [
		new web3._extend.Method({
			name: 'digest',
			call: 'identity_digest',
			params: 1
		}),
	],
	properties: [
		new web3._extend.Property({
			name: 'corexVersion',
			getter: 'identity_corexVersion'
		}),
	]
});
`

const CommunicationJs = `
web3._extend({
	property: 'communication',
	methods: [],
	properties: [
		new web3._extend.Property({
			name: 'version',
			getter: 'communication_version'
		}),
		new web3._extend.Property({
			name: 'listening',
			getter: 'communication_listening'
		}),
		new web3._extend.Property({
			name: 'peerCount',
			getter: 'communication_peerCount'
		}),
	]
});
`

const PoVJs = `
web3._extend({
	property: 'pov',
	methods: [
		new web3._extend.Method({
			name: 'summary',
			call: 'pov_summary',
			params: 0
		}),
		new web3._extend.Method({
			name: 'communication',
			call: 'pov_communication',
			params: 0
		}),
		new web3._extend.Method({
			name: 'governance',
			call: 'pov_governance',
			params: 0
		}),
		new web3._extend.Method({
			name: 'trust',
			call: 'pov_trust',
			params: 0
		}),
		new web3._extend.Method({
			name: 'asset',
			call: 'pov_asset',
			params: 0
		}),
		new web3._extend.Method({
			name: 'nodeRole',
			call: 'pov_nodeRole',
			params: 0
		}),
	],
	properties: []
});
`

const TrustJs = `
web3._extend({
	property: 'trust',
	methods: [
		new web3._extend.Method({
			name: 'getHeaderByNumber',
			call: 'trust_getHeaderByNumber',
			params: 1,
			inputFormatter: [web3._extend.formatters.inputBlockNumberFormatter]
		}),
		new web3._extend.Method({
			name: 'getHeaderByHash',
			call: 'trust_getHeaderByHash',
			params: 1
		}),
		new web3._extend.Method({
			name: 'getBlockByNumber',
			call: 'trust_getBlockByNumber',
			params: 2,
			inputFormatter: [web3._extend.formatters.inputBlockNumberFormatter, function (val) { return !!val; }]
		}),
		new web3._extend.Method({
			name: 'getBlockByHash',
			call: 'trust_getBlockByHash',
			params: 2,
			inputFormatter: [null, function (val) { return !!val; }]
		}),
	],
	properties: []
});
`

const AssetJs = `
web3._extend({
	property: 'asset',
	methods: [
		new web3._extend.Method({
			name: 'submitValueFlow',
			call: 'asset_submitValueFlow',
			params: 1,
			inputFormatter: [web3._extend.formatters.inputTransactionFormatter]
		}),
		new web3._extend.Method({
			name: 'fillValueFlow',
			call: 'asset_fillValueFlow',
			params: 1,
			inputFormatter: [web3._extend.formatters.inputTransactionFormatter]
		}),
		new web3._extend.Method({
			name: 'getValueFlowByHash',
			call: 'asset_getValueFlowByHash',
			params: 1
		}),
		new web3._extend.Method({
			name: 'getValueFlowReceipt',
			call: 'asset_getValueFlowReceipt',
			params: 1
		}),
	],
	properties: []
});
`

const AccountJs = `
web3._extend({
	property: 'account',
	methods: [
		new web3._extend.Method({
			name: 'createAccount',
			call: 'account_createAccount',
			params: 1
		}),
		new web3._extend.Method({
			name: 'authorizeAccount',
			call: 'account_authorizeAccount',
			params: 3,
			inputFormatter: [web3._extend.formatters.inputAddressFormatter, null, null]
		}),
		new web3._extend.Method({
			name: 'revokeAccount',
			call: 'account_revokeAccount',
			params: 1,
			inputFormatter: [web3._extend.formatters.inputAddressFormatter]
		}),
		new web3._extend.Method({
			name: 'submitValueFlow',
			call: 'account_submitValueFlow',
			params: 2,
			inputFormatter: [web3._extend.formatters.inputTransactionFormatter, null]
		}),
	],
	properties: [
		new web3._extend.Property({
			name: 'participants',
			getter: 'account_participants'
		}),
		new web3._extend.Property({
			name: 'managedWallets',
			getter: 'account_managedWallets'
		}),
	]
});
`

const RpcJs = `
web3._extend({
	property: 'rpc',
	methods: [],
	properties: [
		new web3._extend.Property({
			name: 'modules',
			getter: 'rpc_modules'
		}),
	]
});
`

const TxpoolJs = `
web3._extend({
	property: 'flowspace',
	methods: [],
	properties:
	[
		new web3._extend.Property({
			name: 'content',
			getter: 'txpool_content'
		}),
		new web3._extend.Property({
			name: 'inspect',
			getter: 'txpool_inspect'
		}),
		new web3._extend.Property({
			name: 'status',
			getter: 'txpool_status',
			outputFormatter: function(status) {
				status.pending = web3._extend.utils.toDecimal(status.pending);
				status.queued = web3._extend.utils.toDecimal(status.queued);
				return status;
			}
		}),
		new web3._extend.Method({
			name: 'contentFrom',
			call: 'txpool_contentFrom',
			params: 1,
		}),
	]
});
`

const ValueFlowJs = `
web3._extend({
	property: 'valueflow',
	methods: [
		new web3._extend.Method({
			name: 'contentFrom',
			call: 'valueflow_contentFrom',
			params: 1,
		}),
		new web3._extend.Method({
			name: 'valueStatus',
			call: 'valueflow_valueFlowStatus',
			params: 0
		}),
	],
	properties:
	[
		new web3._extend.Property({
			name: 'content',
			getter: 'valueflow_content'
		}),
		new web3._extend.Property({
			name: 'inspect',
			getter: 'valueflow_inspect'
		}),
		new web3._extend.Property({
			name: 'status',
			getter: 'valueflow_valueFlowStatus',
			outputFormatter: function(status) {
				status.active = web3._extend.utils.toDecimal(status.active);
				status.queued = web3._extend.utils.toDecimal(status.queued);
				return status;
			}
		}),
	]
});
`

const LESJs = `
web3._extend({
	property: 'les',
	methods:
	[
		new web3._extend.Method({
			name: 'getCheckpoint',
			call: 'les_getCheckpoint',
			params: 1
		}),
		new web3._extend.Method({
			name: 'clientInfo',
			call: 'les_clientInfo',
			params: 1
		}),
		new web3._extend.Method({
			name: 'priorityClientInfo',
			call: 'les_priorityClientInfo',
			params: 3
		}),
		new web3._extend.Method({
			name: 'setClientParams',
			call: 'les_setClientParams',
			params: 2
		}),
		new web3._extend.Method({
			name: 'setDefaultParams',
			call: 'les_setDefaultParams',
			params: 1
		}),
		new web3._extend.Method({
			name: 'addBalance',
			call: 'les_addBalance',
			params: 2
		}),
	],
	properties:
	[
		new web3._extend.Property({
			name: 'latestCheckpoint',
			getter: 'les_latestCheckpoint'
		}),
		new web3._extend.Property({
			name: 'checkpointContractAddress',
			getter: 'les_getCheckpointContractAddress'
		}),
		new web3._extend.Property({
			name: 'serverInfo',
			getter: 'les_serverInfo'
		}),
	]
});
`

const VfluxJs = `
web3._extend({
	property: 'vflux',
	methods:
	[
		new web3._extend.Method({
			name: 'distribution',
			call: 'vflux_distribution',
			params: 2
		}),
		new web3._extend.Method({
			name: 'timeout',
			call: 'vflux_timeout',
			params: 2
		}),
		new web3._extend.Method({
			name: 'value',
			call: 'vflux_value',
			params: 2
		}),
	],
	properties:
	[
		new web3._extend.Property({
			name: 'requestStats',
			getter: 'vflux_requestStats'
		}),
	]
});
`

const DevJs = `
web3._extend({
	property: 'dev',
	methods:
	[
		new web3._extend.Method({
			name: 'addWithdrawal',
			call: 'dev_addWithdrawal',
			params: 1
		}),
		new web3._extend.Method({
			name: 'setFeeRecipient',
			call: 'dev_setFeeRecipient',
			params: 1
		}),
	],
});
`
