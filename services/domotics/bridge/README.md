# bridge

This package offers a unified, gRPC-defined contract to any number of specific home automation technologies. The goal is to be able to interfact with home and building control devices in a consistent, technology-independent way. This is done by defining a few foundational primitives and exposing ways to interact with these primitives. These are:
 - a bridge
 - one or more devices controlled by the bridge
The bridge is usually a physical entity which connects to a computer (or network), and acts as a gateway to the device or the home automation network which the devices connect to.

The bridge and device contracts both have distinct 'config' and 'state' elements. The 'config' element contains parameters which can be used to manipulate metadata associated with the parent, while the 'state' element contains parameters which manipulate the state of the parent itself. The intent of each object is that it is self-describing; it is possible for any consumer of the contract to understand what can be done to the device by reading properties directly from the element which describe the possible values of the different state elements.

There exist a number of implementations of this contract today included here. This package includes the contract definition, along with a couple of helper service definitions which can be included by particular implementations as needed.

The bridge contract is designed to support both major protocol approaches:
 1. those technologies and SDKs which expose a synchronous, request/response style interface.
 2. those technologies and SDKs which expose an asynchronous, event style interface.

There are a few foundational principles which implementers of this contract should keep in mind.
 1. performing a set action on an element should cause the resulting state value to be propagated to any subscribed clients via the 'Update' stream. Some protocols (such as ZWave, Deconz, etc.) cause this to happen automatically; while more low-level protocols (such as X10) may require that the implementation perform this manually. Handling of this is provided automatically by the SyncBridgeService.
 2. consumers of the contract should be able to assume the device ID is the one true identity of a device; and should it migrate between bridges the device itself will not change. As a result, clients of the contract will not intrinsically link devices to bridges outside the active connection between the client and the bridge.

Bridges advertise themselves over SSDP, using the `falnet_nerves:bridge` type. Typically advertisements are sent every 30 seconds.
