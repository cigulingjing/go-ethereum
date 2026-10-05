In Fig. 1, we show a comprehensive system model for our
proposed LEAGAN. The blockchain consists of a peer-to-peer
network. Any peer has the right to notify any legitimate error
or bug in the smart contract. Besides, we also open the scope
of the upgradeable smart contract not for error or bug fixes, but
LEAGAN also allows to include a verified change in the form of
any program condition or statement in the smart contract. Thus,
LEAGAN becomes a novel generic framework for blockchain
upgradeable smart contracts. Any peer (change requester) requesting a change broadcasts a change request message in the
blockchain network. The other peers in the blockchain network
verify the signature of the requester and the change request. We
segregate the smart contract composition into three parts: business, data, and status. We can also add the term contract to each
of them to maintain the notion of automated transactions. In an
upgradable smart contract scenario, the logic contract contains
the business logic (mutable), the data storage holds the contract’s
state and data (immutable), and the status contract tracks the
version and update status (mutable) of the logic contract. When
an upgrade occurs, the logic contract is updated while the data
storage remains unchanged, and the status contract records the
new version; this ensures continuity and transparency, i.e., nonhalt deployment. Once the signature and the change required
parameters are verified, LEAGAN uses an incremental hash to
update the business logic in the smart contract. At the same time,
LEAGAN uses a revision control system to manage the versions
of the smart contract and updates the address of the business logic
in the data storage. This mechanism is integrated directly into the
LEAGAN framework, functioning as part of the smart contract
system rather than as a separate application. It is designed to
seamlessly operate within the blockchain environment, ensuring
that updates are decentralized and transparent. We discuss all the
details of the above-mentioned functionalities in Section III-B