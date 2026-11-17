package trip

type TripRepository struct{ pool database }

func NewTripRepository(pool database) *TripRepository { return &TripRepository{pool: pool} }
