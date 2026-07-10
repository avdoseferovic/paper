package entity

import (
	"fmt"
	"time"
)

// FileAttachment describes a file embedded in the generated PDF.
type FileAttachment struct {
	// FileName is the name shown by PDF viewers and stored in the file
	// specification dictionary.
	FileName string
	// MIMEType is written as the embedded file stream subtype. It defaults to
	// application/octet-stream when empty.
	MIMEType string
	// Description is an optional human-readable file description.
	Description string
	// AFRelationship is the PDF associated-file relationship name. It defaults
	// to Unspecified when empty.
	AFRelationship string
	// Data is the raw file content to embed.
	Data []byte
	// CreationDate is written to the embedded file parameters. If zero, the
	// document creation date or generation time is used.
	CreationDate time.Time
}

func appendFileAttachmentsMap(attachments []FileAttachment, m map[string]any) map[string]any {
	for i, attachment := range attachments {
		prefix := fmt.Sprintf("config_file_attachment_%d", i)
		if attachment.FileName != "" {
			m[prefix+"_filename"] = attachment.FileName
		}
		if attachment.MIMEType != "" {
			m[prefix+"_mime_type"] = attachment.MIMEType
		}
		if attachment.Description != "" {
			m[prefix+"_description"] = attachment.Description
		}
		if attachment.AFRelationship != "" {
			m[prefix+"_af_relationship"] = attachment.AFRelationship
		}
		m[prefix+"_size"] = len(attachment.Data)
		if !attachment.CreationDate.IsZero() {
			m[prefix+"_creation_date"] = attachment.CreationDate
		}
	}
	return m
}

// CloneAttachments returns an independent copy of attachments, including the
// embedded data bytes.
func CloneAttachments(attachments []FileAttachment) []FileAttachment {
	if len(attachments) == 0 {
		return nil
	}
	clones := make([]FileAttachment, len(attachments))
	for i, attachment := range attachments {
		attachment.Data = append([]byte(nil), attachment.Data...)
		clones[i] = attachment
	}
	return clones
}
