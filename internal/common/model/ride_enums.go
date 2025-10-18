package model

type EntityType string

const (
	EntityTypeDriver    EntityType = "driver"
	EntityTypePassenger EntityType = "passenger"
)

type Role string

const (
	RolePassenger Role = "PASSENGER"
	RoleDriver    Role = "DRIVER"
	RoleAdmin     Role = "ADMIN"
)

type UserStatus string

const (
	UserActive   UserStatus = "ACTIVE"
	UserInactive UserStatus = "INACTIVE"
	UserBanned   UserStatus = "BANNED"
)

type RideStatus string

const (
	RideRequested  RideStatus = "REQUESTED"
	RideMatched    RideStatus = "MATCHED"
	RideEnRoute    RideStatus = "EN_ROUTE"
	RideArrived    RideStatus = "ARRIVED"
	RideInProgress RideStatus = "IN_PROGRESS"
	RideCompleted  RideStatus = "COMPLETED"
	RideCancelled  RideStatus = "CANCELLED"
)

type VehicleType string

const (
	VehicleEconomy VehicleType = "ECONOMY"
	VehiclePremium VehicleType = "PREMIUM"
	VehicleXL      VehicleType = "XL"
)

type RideEventType string

const (
	EventRideRequested   RideEventType = "RIDE_REQUESTED"
	EventDriverMatched   RideEventType = "DRIVER_MATCHED"
	EventDriverArrived   RideEventType = "DRIVER_ARRIVED"
	EventRideStarted     RideEventType = "RIDE_STARTED"
	EventRideCompleted   RideEventType = "RIDE_COMPLETED"
	EventRideCancelled   RideEventType = "RIDE_CANCELLED"
	EventStatusChanged   RideEventType = "STATUS_CHANGED"
	EventLocationUpdated RideEventType = "LOCATION_UPDATED"
	EventFareAdjusted    RideEventType = "FARE_ADJUSTED"
)
