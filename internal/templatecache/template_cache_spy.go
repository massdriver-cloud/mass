package templatecache

/*
Simple test spy created to defer picking a Mocking framework.
The commands were so simple we just need to make sure the right thing is called.
*/
type CallTracker struct {
	Calls int
}

func (c *CallTracker) AddCall() {
	c.Calls++
}

type MockCacheClient struct {
	Calls map[string]*CallTracker
}

func (b *MockCacheClient) ListTemplates() ([]string, error) {
	b.addOrCreateTracker("ListTemplates")

	return []string{"applications/aws-lambda", "infrastructure/terraform"}, nil
}

func (b *MockCacheClient) RefreshTemplates() error {
	b.addOrCreateTracker("RefreshTemplates")

	return nil
}

func (b *MockCacheClient) GetTemplatePath() (string, error) {
	b.addOrCreateTracker("GetTemplatePath")

	return "/home/", nil
}

func (b *MockCacheClient) addOrCreateTracker(calledFunction string) {
	val, exists := b.Calls[calledFunction]

	if exists {
		val.AddCall()
	} else {
		b.Calls[calledFunction] = &CallTracker{Calls: 1}
	}
}
