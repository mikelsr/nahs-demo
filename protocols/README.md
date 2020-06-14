# Protocols

## [Bike rental](./bike_rental.bspl)

#### Roles

* Client
* Renter

#### Parameters

* `ID`: Identifier of the instance.
* `bikeID`: Identifier of the bike to be rented.
* `price`: Accored price in euros per minute.
* `origin`: ID of the bike station the bike is being rent at.
* `destination`: ID of the bike station the bike is being dropped at, can be empty if the destination is unknown.
* `rID`: either accept or reject.

#### Actions

`Client -> Renter: request[in origin, in destination, out ID]`

Request a bike.

`Renter -> Client: offer[in ID, in origin, in destination, out bikeID, out price]`

Offer a bike to a client for a price.

`Client -> Renter: accept[in ID, in bikeID, in price, out rID]`

Accept the offer.

`Client -> Renter: reject[in ID, in bikeID, in price, out rID]`

Reject the offer.
