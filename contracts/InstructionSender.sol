// SPDX-License-Identifier: MIT
pragma solidity ^0.8.27;

// TODO: Replace local interfaces with imports from flare-smart-contracts-v2 once published as a package.
import { ITeeExtensionRegistry } from "./interfaces/ITeeExtensionRegistry.sol";
import { ITeeMachineRegistry } from "./interfaces/ITeeMachineRegistry.sol";

/// @title HelloWorldInstructionSender
/// @author Flare Foundation
/// @notice Hello World example — on-chain entry point for sending instructions to the TEE.
///
/// DO NOT MODIFY: constructor, setExtensionId(), _getExtensionId(), OP_COMMAND_PLACEHOLDER
contract HelloWorldInstructionSender {
    /// @notice Operation type for the SAY_HELLO action.
    bytes32 public constant OP_TYPE_SAY_HELLO = bytes32("SAY_HELLO");

    // --- DO NOT MODIFY ---
    /// @notice Placeholder command field passed with every instruction.
    bytes32 public constant OP_COMMAND_PLACEHOLDER = bytes32("PLACEHOLDER");

    /// @notice Reference to the TEE extension registry contract.
    ITeeExtensionRegistry public immutable teeExtensionRegistry;
    /// @notice Reference to the TEE machine registry contract.
    ITeeMachineRegistry public immutable teeMachineRegistry;

    uint256 private _extensionId;

    /// @notice Initializes the contract with registry addresses.
    /// @param _teeExtensionRegistry Address of the TEE extension registry.
    /// @param _teeMachineRegistry Address of the TEE machine registry.
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

    /// @notice Sends a SAY_HELLO instruction to the TEE.
    /// @param _message JSON-encoded payload (e.g. {"name": "Alice"}).
    function sendSayHello(bytes calldata _message) external payable {
        address[] memory teeIds = teeMachineRegistry.getRandomTeeIds(_getExtensionId(), 1);
        address[] memory cosigners = new address[](0);
        uint64 cosignersThreshold = 0;

        teeExtensionRegistry.sendInstructions{value: msg.value}(
            teeIds,
            OP_TYPE_SAY_HELLO,
            OP_COMMAND_PLACEHOLDER,
            _message,
            cosigners,
            cosignersThreshold
        );
    }

    /// @notice Returns the cached extension ID, reverting if not yet set.
    /// @return The extension ID assigned to this contract.
    function _getExtensionId() internal view returns (uint256) {
        require(_extensionId != 0, "Extension ID is not set.");
        return _extensionId;
    }
}
