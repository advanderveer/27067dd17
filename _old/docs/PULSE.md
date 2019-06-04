# Pulse

we create a chain, each block describes a sorted list of all members that is supposed to make
a block. Whenever he gets it his proposal block will be broadcasted immediatly.
Everyone keeps a timer, if it expires they will accept blocks from the next member
in the list. If it also doesn't provide one the next one will be picked, etc etc.

It can happen that a part of the network doesn't receive a block on time while
another part does. As long as honest members only pulse for one of the tips
one should progress faster then the other.

The public key must be past a certain threshold, such that it takes way longer to
generate then a single time slot. So it doesn't make it worth while to create identities
to quickly create pks that win the next slot.

Every block can add or remove new members to the list. Members that repeatedly
failed to respond have their stake reset.

Every message prescribes the next slots of members to pulse. Basic slot leader
election over UDP.

From receival of the message we only accept a message from the next member in N
seconds, from another member after that, from another member after that, etc etc

Next member is selected randomnly through vrf.  

we only accept public keys of a certain strength to to defer sybill attacks

what happens if half of the network got the pulse in time, while the other didn't?
(1) the timeout value should be chosen such that that very unlikely
(2) but what of a network segmentation?

- _How about when a serie of nodes doesn't respond, will it pause for a long time?_

# idead 2:

What if the two blocks are the same "prev" if they have the same top-N tokens?
