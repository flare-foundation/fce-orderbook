// SPDX-License-Identifier: MIT
pragma solidity ^0.8.27;

import { ITeeExtensionRegistry } from "flare-smart-contracts-v2/contracts/userInterfaces/tee/ITeeExtensionRegistry.sol";
import { ITeeMachineRegistry } from "flare-smart-contracts-v2/contracts/userInterfaces/tee/ITeeMachineRegistry.sol";

/// @title MyExtensionInstructionSender
/// @notice On-chain entry point for sending instructions to your extension's TEE.
///
/// HOW TO CUSTOMIZE:
///   1. Rename the contract to match your extension (e.g., OrderbookInstructionSender)
///   2. Define your OP_TYPE constants — one per action your extension handles
///   3. Add one send function per action type (follow the sendMyInstruction pattern)
///   4. After modifying, run: ./scripts/generate-bindings.sh
///
/// DO NOT MODIFY: constructor, setExtensionId(), _getExtensionId(), OP_COMMAND_PLACEHOLDER
contract MyExtensionInstructionSender {
    uint256 _extensionId;

    ITeeExtensionRegistry public immutable teeExtensionRegistry;
    ITeeMachineRegistry public immutable teeMachineRegistry;

    // --- CUSTOMIZE: Define your operation types ---
    // Each OP_TYPE maps to a handler in your Go extension's processAction().
    // The bytes32 value here must match teeutils.ToHash("MY_ACTION") on the Go side.
    bytes32 constant OP_TYPE_MY_ACTION = bytes32("MY_ACTION");
    // bytes32 constant OP_TYPE_ANOTHER_ACTION = bytes32("ANOTHER_ACTION");

    // --- DO NOT MODIFY ---
    bytes32 constant OP_COMMAND_PLACEHOLDER = bytes32("PLACEHOLDER");

    constructor(
        ITeeExtensionRegistry _teeExtensionRegistry,
        ITeeMachineRegistry _teeMachineRegistry
    ) {
        teeExtensionRegistry = _teeExtensionRegistry;
        teeMachineRegistry = _teeMachineRegistry;
    }

    /// @notice Finds and sets this contract's extension id. Can only be set once.
    /// DO NOT MODIFY this function.
    function setExtensionId() external {
        require(_extensionId == 0, "Extension ID already set.");

        uint256 c = teeExtensionRegistry.extensionsCounter();
        for (uint256 i = 0; i < c; ++i) {
            if (teeExtensionRegistry.getTeeExtensionInstructionsSender(i) == address(this)) {
                _extensionId = i;
                return;
            }
        }
        revert("Extension ID not found.");
    }

    /// @notice CUSTOMIZE: Rename and add send functions for each action type.
    /// @param _message JSON-encoded payload matching your extension's expected format.
    function sendMyInstruction(bytes calldata _message) external payable {
        address[] memory teeIds = teeMachineRegistry.getRandomTeeIds(_getExtensionId(), 1);
        address[] memory cosigners = new address[](0);
        uint64 cosignersThreshold = 0;

        teeExtensionRegistry.sendInstructions{value: msg.value}(
            teeIds,
            OP_TYPE_MY_ACTION,
            OP_COMMAND_PLACEHOLDER,
            _message,
            cosigners,
            cosignersThreshold
        );
    }

    function _getExtensionId() internal view returns (uint256) {
        require(_extensionId != 0, "Extension ID is not set.");
        return _extensionId;
    }
}
