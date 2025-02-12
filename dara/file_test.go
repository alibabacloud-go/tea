package dara

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

// TestNewDaraFile tests the NewDaraFile function
func TestNewDaraFile(t *testing.T) {
	path := "testfile.txt"
	tf := NewDaraFile(path)
	if tf.Path() != path {
		t.Errorf("Expected path to be %s, got %s", path, tf.Path())
	}
}

// TestCreateTime tests the CreateTime method
func TestCreateTime(t *testing.T) {
	path := "testfile.txt"
	ioutil.WriteFile(path, []byte("test"), 0644) // 创建文件以确保它存在
	defer os.Remove(path)

	tf := NewDaraFile(path)
	date, err := tf.CreateTime()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if date == nil {
		t.Error("expected a valid TeaDate, got nil")
	}
}

// TestModifyTime tests the ModifyTime method
func TestModifyTime(t *testing.T) {
	path := "testfile.txt"
	ioutil.WriteFile(path, []byte("test"), 0644)
	defer os.Remove(path)

	tf := NewDaraFile(path)
	date, err := tf.ModifyTime()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if date == nil {
		t.Error("expected a valid TeaDate, got nil")
	}
}

// TestLength tests the Length method
func TestLength(t *testing.T) {
	path := "testfile.txt"
	content := []byte("Hello, World!")
	ioutil.WriteFile(path, content, 0644)
	defer os.Remove(path)

	tf := NewDaraFile(path)
	length, err := tf.Length()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if length != int64(len(content)) {
		t.Errorf("expected length %d, got %d", len(content), length)
	}
}

// TestRead tests the Read method
func TestRead(t *testing.T) {
	path := "testfile.txt"
	content := []byte("Hello, World!")
	ioutil.WriteFile(path, content, 0644)
	defer os.Remove(path)

	tf := NewDaraFile(path)
	data, err := tf.Read(5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != "Hello" {
		t.Errorf("expected 'Hello', got '%s'", string(data))
	}

	// Read the rest of the file
	data, err = tf.Read(10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != ", World!" {
		t.Errorf("expected ', World!', got '%s'", string(data))
	}
}

// TestWrite tests the Write method
func TestWrite(t *testing.T) {
	path := "testfile.txt"
	tf := NewDaraFile(path)

	data := []byte("Hello, Write Test!")
	err := tf.Write(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Validate the content of the file
	readData, _ := ioutil.ReadFile(path)
	if string(readData) != "Hello, Write Test!" {
		t.Errorf("expected file content to be %s, got %s", string(data), string(readData))
	}
}

// TestClose tests the Close method
func TestClose(t *testing.T) {
	path := "testfile.txt"
	tf := NewDaraFile(path)
	err := tf.Close()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestExists tests the Exists function
func TestExists(t *testing.T) {
	path := "testfile.txt"
	ioutil.WriteFile(path, []byte("test"), 0644)
	defer os.Remove(path)

	exists, err := Exists(path)
	if err != nil || !exists {
		t.Errorf("expected file to exist, got error: %v", err)
	}

	exists, err = Exists("nonexistent.txt")
	if err != nil || exists {
		t.Errorf("expected file to not exist, got error: %v", err)
	}
}

func TestCreateReadWriteStream(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "example")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)
	testFile := filepath.Join(tempDir, "test.txt")
	testWFile := filepath.Join(tempDir, "test2.txt")

	// Prepare the test file
	originalContent := "Hello, World!"
	if err := ioutil.WriteFile(testFile, []byte(originalContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Test CreateReadStream
	rs, err := CreateReadStream(testFile)
	if err != nil {
		t.Fatalf("failed to create read stream: %v", err)
	}
	defer rs.Close()

	// Test CreateWriteStream
	ws, err := CreateWriteStream(testWFile)
	if err != nil {
		t.Fatalf("failed to create write stream: %v", err)
	}
	defer ws.Close()

	// Pipe data from read stream to write stream
	if _, err := io.Copy(ws, rs); err != nil {
		t.Fatalf("failed to copy data from read stream to write stream: %v", err)
	}

	// Read back the content to check if it's correct
	data, err := ioutil.ReadFile(testWFile)
	if err != nil {
		t.Fatalf("failed to read back test file: %v", err)
	}

	if string(data) != originalContent {
		t.Fatalf("expected %q but got %q", originalContent, string(data))
	}
}
