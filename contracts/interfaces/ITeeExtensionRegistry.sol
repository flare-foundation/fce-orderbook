// SPDX-License-Identifier: MIT
pragma solidity >=0.7.6 <0.9;

// TODO: Replace this minimal interface with the full import once flare-smart-contracts-v2
// is published as a package:
//   import { ITeeExtensionRegistry } from "flare-smart-contracts-v2/contracts/userInterfaces/tee/ITeeExtensionRegistry.sol";
interface ITeeExtensionRegistry {
    function sendInstructions(
        address[] memory _teeIds,
        bytes32 _opType,
        bytes32 _opCommand,
        bytes memory _message,
        address[] memory _cosigners,
        uint64 _cosignersThreshold
    ) external payable returns (bytes32 _instructionId);

    function extensionsCounter() external view returns (uint256);

    function getTeeExtensionInstructionsSender(uint256 _extensionId)
        external view returns (address);
}
