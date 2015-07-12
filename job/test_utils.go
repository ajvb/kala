package job

import "time"

type MockDB struct{}

func (m *MockDB) GetAll() ([]*Job, error) {
	return nil, nil
}
func (m *MockDB) Get(id string) (*Job, error) {
	return nil, nil
}
func (m *MockDB) Delete(id string) {}
func (m *MockDB) Save(job *Job) error {
	return nil
}
func (m *MockDB) Close() {}

func NewMockCache() *MemoryJobCache {
	return NewMemoryJobCache(&MockDB{}, time.Hour*5)
}
