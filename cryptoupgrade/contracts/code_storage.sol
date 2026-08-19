// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import "./utils/ownable.sol";

contract CodeStorage{
    // Code should bind concrete gas
    struct algo{
        string code;
        uint64 gas;
        string itype; // algorithm's parameter type, 10 char/B
        string otype; // algorithm's return type
        uint64 version;
        uint64 activationBlock;
        bool exists;
    }
    // Algorithm probable corrspond mutiple implementations 
    mapping(string => algo) private Algos;
    mapping(string => mapping(uint64 => algo)) private AlgoVersions;
    mapping(string => uint64[]) private AlgoVersionList;
    string[] private codeNames;
    // Index will make params store in topics as Hash, which will make parse impossible 
    event codeUploaded(string name);
    event codeVersionUploaded(string name, uint64 version, uint64 activationBlock);

    function uploadCode(string memory name,string memory code,uint64 gas, string memory itype,string memory otype) external  {
        _uploadCodeVersion(name, 1, code, gas, itype, otype, 0, true);
    }

    function uploadCodeVersion(string memory name,uint64 version,string memory code,uint64 gas,string memory itype,string memory otype,uint64 activationBlock) external {
        _uploadCodeVersion(name, version, code, gas, itype, otype, activationBlock, false);
    }

    function uploadCodeImmediate(string memory name,uint64 version,string memory code,uint64 gas,string memory itype,string memory otype) external {
        _uploadCodeVersion(name, version, code, gas, itype, otype, uint64(block.number), false);
    }

    function _uploadCodeVersion(string memory name,uint64 version,string memory code,uint64 gas,string memory itype,string memory otype,uint64 activationBlock,bool legacyEvent) internal {
        require(bytes(code).length > 0, "Code cannot be empty"); 
        require(gas>0,"gas is less than 0");
        require(version>0,"version is zero");
        require(!AlgoVersions[name][version].exists,"algorithm version already exist");

        if (AlgoVersionList[name].length==0) {
            codeNames.push(name);
        }
        AlgoVersions[name][version]=algo(code,gas,itype,otype,version,activationBlock,true);
        AlgoVersionList[name].push(version);
        if (Algos[name].gas==0 || activationBlock<=uint64(block.number)) {
            Algos[name]=AlgoVersions[name][version];
        }
        if (legacyEvent) {
            emit codeUploaded(name);
        } else {
            emit codeVersionUploaded(name,version,activationBlock);
        }
    }

    function updataGas(string memory name,uint64 _gas) external {
        require(bytes(Algos[name].code).length>0,"Code is empty");
        require(_gas>0,"gas is less than 0");
        Algos[name].gas=_gas;
    }

    // This function implement is empty, only provide an gateway to call algorithm.
    function callFunc(string memory name, bytes memory input) external view returns(bytes memory){
        algo memory current = _activeAlgo(name);
        require(current.gas>0,"algorithm not exist");
    }

    // Return value must have name, geth use these to decode output.
    function getInfo(string memory name) external view returns(string memory code,uint64 gas, string memory itype,string memory otype){
        algo memory current = _activeAlgo(name);
        return (current.code,current.gas,current.itype,current.otype);
    }

    function getVersionInfo(string memory name,uint64 version) external view returns(string memory code,uint64 gas,string memory itype,string memory otype,uint64 storedVersion,uint64 activationBlock){
        algo memory item = AlgoVersions[name][version];
        return (item.code,item.gas,item.itype,item.otype,item.version,item.activationBlock);
    }

    function getActiveVersion(string memory name) external view returns(uint64 version,uint64 activationBlock){
        algo memory current = _activeAlgo(name);
        return (current.version,current.activationBlock);
    }

    function getActiveInfo(string memory name) external view returns(string memory code,uint64 gas,string memory itype,string memory otype,uint64 version,uint64 activationBlock){
        algo memory current = _activeAlgo(name);
        return (current.code,current.gas,current.itype,current.otype,current.version,current.activationBlock);
    }

    function getGas(string memory name) external view returns(uint64){
        return _activeAlgo(name).gas;
    }

    function getAllAlgo() external view returns(string[] memory){
        return codeNames;
    }

    function _activeAlgo(string memory name) internal view returns(algo memory){
        uint64[] memory versions = AlgoVersionList[name];
        algo memory selected;
        for (uint i=0; i<versions.length; i++) {
            algo memory item = AlgoVersions[name][versions[i]];
            if (item.exists && item.activationBlock<=uint64(block.number) && item.version>=selected.version) {
                selected = item;
            }
        }
        return selected;
    }
}
