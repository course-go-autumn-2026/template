package finish

import (
	api "github.com/xotapy-main/courseGo/internal/generated"
	"github.com/xotapy-main/courseGo/internal/http/trip/create"
	"github.com/xotapy-main/courseGo/internal/model"
)

func toAPITrip(t *model.Trip) api.Trip {
	return create.ToAPITrip(t)
}
