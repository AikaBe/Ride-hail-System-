package model

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
