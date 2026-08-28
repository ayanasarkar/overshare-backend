// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract OvershareCertificate {
    struct Certificate {
        bytes32 imageHash;
        string ipfsCID;
        uint256 timestamp;
    }
    mapping(uint256 => Certificate) public certificates;
    uint256 public nextId;

    event CertificateMinted(uint256 indexed id, bytes32 imageHash, string ipfsCID);

    function mintCertificate(bytes32 imageHash, string calldata ipfsCID) external returns (uint256) {
        uint256 id = nextId++;
        certificates[id] = Certificate(imageHash, ipfsCID, block.timestamp);
        emit CertificateMinted(id, imageHash, ipfsCID);
        return id;
    }
}
