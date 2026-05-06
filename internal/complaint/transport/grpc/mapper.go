package grpc

import (
	complaintv1 "github.com/go-park-mail-ru/2026_1_ASAP/gen/go/complaint/v1"
	dtoComplaint "github.com/go-park-mail-ru/2026_1_ASAP/internal/complaint/dto/complaint"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func mapComplaintTypeToString(t complaintv1.ComplaintType) string {
	switch t {
	case complaintv1.ComplaintType_COMPLAINT_TYPE_BUG:
		return "bug"
	case complaintv1.ComplaintType_COMPLAINT_TYPE_PRODUCT:
		return "product"
	case complaintv1.ComplaintType_COMPLAINT_TYPE_UPGRADE:
		return "upgrade"
	default:
		return ""
	}
}

func mapComplaintStatusToString(s complaintv1.ComplaintStatus) string {
	switch s {
	case complaintv1.ComplaintStatus_COMPLAINT_STATUS_NEW:
		return "new"
	case complaintv1.ComplaintStatus_COMPLAINT_STATUS_IN_PROGRESS:
		return "in_progress"
	case complaintv1.ComplaintStatus_COMPLAINT_STATUS_CLOSED:
		return "closed"
	default:
		return ""
	}
}

func mapComplaintToProto(c dtoComplaint.ComplaintDTO) *complaintv1.ComplaintItem {
	item := &complaintv1.ComplaintItem{
		Id:     c.ID,
		Type:   c.Type,
		Status: c.Status,
		Feedback: &complaintv1.Feedback{
			FeedbackName:  c.FeedbackDTO.FeedBackName,
			FeedbackEmail: c.FeedbackDTO.FeedBackEmail,
		},
		Body:      c.Body,
		UserId:    c.UserID,
		CreatedAt: timestamppb.New(c.CreatedAt),
		UpdatedAt: timestamppb.New(c.UpdatedAt),
	}
	if c.AttachmentURL != nil {
		item.AttachmentUrl = c.AttachmentURL
	}
	return item
}

func mapComplaintsToProto(list []dtoComplaint.ComplaintDTO) []*complaintv1.ComplaintItem {
	result := make([]*complaintv1.ComplaintItem, 0, len(list))
	for _, c := range list {
		result = append(result, mapComplaintToProto(c))
	}
	return result
}
