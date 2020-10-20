package document

import "time"

func NewBuilder() DocumentBuilder {
	return &documentBuilder{}
}

type document struct {
	id           int64
	uuid         string
	name         string
	content      string
	created      time.Time
	lastModified time.Time
}

type Document interface {
	Id() int64
	Uuid() string
	Name() string
	Content() string
	Created() time.Time
	LastModified() time.Time
}

func (d *document) Id() int64 {
	return d.id
}

func (d *document) Uuid() string {
	return d.uuid
}

func (d *document) Name() string {
	return d.name
}

func (d *document) Content() string {
	return d.content
}

func (d *document) Created() time.Time {
	return d.created
}

func (d *document) LastModified() time.Time {
	return d.lastModified
}

type documentBuilder struct {
	id           int64
	uuid         string
	name         string
	content      string
	created      time.Time
	lastModified time.Time
}

type DocumentBuilder interface {
	WithId(int64) DocumentBuilder
	WithUuid(string) DocumentBuilder
	WithName(string) DocumentBuilder
	WithContent(string) DocumentBuilder
	WithCreated(time.Time) DocumentBuilder
	WithLastModified(time.Time) DocumentBuilder
	Build() Document
}

func (b *documentBuilder) WithId(id int64) DocumentBuilder {
	b.id = id
	return b
}

func (b *documentBuilder) WithUuid(uuid string) DocumentBuilder {
	b.uuid = uuid
	return b
}

func (b *documentBuilder) WithName(name string) DocumentBuilder {
	b.name = name
	return b
}

func (b *documentBuilder) WithContent(content string) DocumentBuilder {
	b.content = content
	return b
}

func (b *documentBuilder) WithCreated(created time.Time) DocumentBuilder {
	b.created = created
	return b
}

func (b *documentBuilder) WithLastModified(lastModified time.Time) DocumentBuilder {
	b.lastModified = lastModified
	return b
}

func (b *documentBuilder) Build() Document {
	return &document{
		id:           b.id,
		uuid:         b.uuid,
		name:         b.name,
		content:      b.content,
		created:      b.created,
		lastModified: b.lastModified,
	}
}
