/* Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package lucy

import "testing"
import "os"
import "reflect"
import "strings"

import "git-wip-us.apache.org/repos/asf/lucy-clownfish.git/runtime/go/clownfish"

func TestIndexerAddDoc(t *testing.T) {
	schema := createTestSchema()
	index := NewRAMFolder("")
	indexer, _ := OpenIndexer(&OpenIndexerArgs{
		Create: true,
		Index:  index,
		Schema: schema,
	})
	indexer.AddDoc(&testDoc{Content: "foo"})
	indexer.AddDoc(map[string]interface{}{"content": "foo"})
	doc := NewDoc(0)
	doc.Store("content", "foo")
	indexer.AddDoc(doc)
	indexer.Commit()
	searcher, _ := OpenIndexSearcher(index)
	if got := searcher.DocFreq("content", "foo"); got != 3 {
		t.Errorf("Didn't index all docs -- DocMax: %d", got)
	}
}

func TestIndexerAddIndex(t *testing.T) {
	var err error
	origIndex := "_test_go_indexer_add_index"
	defer os.RemoveAll(origIndex)
	schema := createTestSchema()
	indexer, err := OpenIndexer(&OpenIndexerArgs{
		Create: true,
		Index:  origIndex,
		Schema: schema,
	})
	if err != nil {
		t.Errorf("OpenIndexer: %v", err)
		return
	}
	err = indexer.AddDoc(&testDoc{Content: "foo"})
	if err != nil {
		t.Errorf("AddDoc: %v", err)
	}
	err = indexer.Commit()
	if err != nil {
		t.Errorf("Commit FS index: %v", err)
	}

	indexer, _ = OpenIndexer(&OpenIndexerArgs{
		Create: true,
		Index:  NewRAMFolder(""),
		Schema: schema,
	})
	err = indexer.AddIndex(origIndex)
	if err != nil {
		t.Errorf("AddIndex: %v", err)
	}
	err = indexer.Commit()
	if err != nil {
		t.Errorf("AddIndex: %v", err)
	}
}

func TestIndexerDeletions(t *testing.T) {
	index := createTestIndex("foo", "bar", "baz", "gazi")
	indexer, _ := OpenIndexer(&OpenIndexerArgs{Index: index})
	err := indexer.DeleteByTerm("content", "foo")
	if err != nil {
		t.Errorf("DeleteByTerm: %v", err)
	}
	indexer.DeleteByQuery(NewTermQuery("content", "bar"))
	if err != nil {
		t.Errorf("DeleteByQuery: %v", err)
	}
	indexer.DeleteByDocID(3)
	if err != nil {
		t.Errorf("DeleteByDocID: %v", err)
	}
	err = indexer.Commit()
	if err != nil {
		t.Errorf("Commit: %v", err)
	}
	searcher, _ := OpenIndexSearcher(index)
	if count := searcher.GetReader().DocCount(); count != 1 {
		t.Errorf("Some deletions didn't go through (count=%d)", count)
	}
}

func TestIndexerMisc(t *testing.T) {
	var err error
	index := createTestIndex("foo", "bar", "baz")
	indexer, _ := OpenIndexer(&OpenIndexerArgs{Index: index})
	if _, ok := indexer.GetSchema().(Schema); !ok {
		t.Errorf("GetSchema")
	}
	if _, ok := indexer.GetStockDoc().(Doc); !ok {
		t.Errorf("GetStockDoc")
	}
	if _, ok := indexer.GetSegWriter().(SegWriter); !ok {
		t.Errorf("GetSegWriter")
	}
	indexer.AddDoc(&testDoc{Content: "gazi"})
	indexer.Optimize()
	err = indexer.PrepareCommit()
	if err != nil {
		t.Errorf("PrepareCommit: %v", err)
	}
	err = indexer.Commit()
	if err != nil {
		t.Errorf("Commit: %v", err)
	}
}

func TestBackgroundMergerMisc(t *testing.T) {
	var err error
	index := createTestIndex("foo", "bar", "baz")
	merger, _ := OpenBackgroundMerger(index, nil)
	merger.Optimize()
	err = merger.PrepareCommit()
	if err != nil {
		t.Errorf("PrepareCommit: %v", err)
	}
	err = merger.Commit()
	if err != nil {
		t.Errorf("Commit: %v", err)
	}
}

func TestIndexManagerAccessors(t *testing.T) {
	host := "dev.example.com"
	manager := NewIndexManager(host, nil)
	if got := manager.GetHost(); got != host {
		t.Errorf("GetHost: %v", got)
	}
	folder := NewRAMFolder("")
	manager.SetFolder(folder)
	if got := manager.GetFolder(); !reflect.DeepEqual(folder, got) {
		t.Errorf("SetFolder/GetFolder")
	}
	manager.SetWriteLockTimeout(72)
	if got := manager.GetWriteLockTimeout(); got != 72 {
		t.Errorf("Set/GetWriteLockTimeout: %d", got)
	}
	manager.SetWriteLockInterval(42)
	if got := manager.GetWriteLockInterval(); got != 42 {
		t.Errorf("Set/GetWriteLockInterval: %d", got)
	}
	manager.SetMergeLockTimeout(73)
	if got := manager.GetMergeLockTimeout(); got != 73 {
		t.Errorf("Set/GetMergeLockTimeout: %d", got)
	}
	manager.SetMergeLockInterval(43)
	if got := manager.GetMergeLockInterval(); got != 43 {
		t.Errorf("Set/GetMergeLockInterval: %d", got)
	}
	manager.SetDeletionLockTimeout(71)
	if got := manager.GetDeletionLockTimeout(); got != 71 {
		t.Errorf("Set/GetDeletionLockTimeout: %d", got)
	}
	manager.SetDeletionLockInterval(41)
	if got := manager.GetDeletionLockInterval(); got != 41 {
		t.Errorf("Set/GetDeletionLockInterval: %d", got)
	}
}

func TestIndexManagerLocks(t *testing.T) {
	manager := NewIndexManager("", nil)
	manager.SetFolder(NewRAMFolder(""))
	if _, ok := manager.MakeWriteLock().(Lock); !ok {
		t.Errorf("MakeWriteLock")
	}
	if _, ok := manager.MakeMergeLock().(Lock); !ok {
		t.Errorf("MakeMergeLock")
	}
	if _, ok := manager.MakeDeletionLock().(Lock); !ok {
		t.Errorf("MakeDeletionLock")
	}
	snapFile := "snapshot_4a.json"
	if _, ok := manager.MakeSnapshotReadLock(snapFile).(SharedLock); !ok {
		t.Errorf("MakeDeletionLock")
	}
}

func TestIndexManagerMergeData(t *testing.T) {
	var err error
	manager := NewIndexManager("", nil)
	manager.SetFolder(NewRAMFolder(""))
	err = manager.WriteMergeData(42)
	if err != nil {
		t.Errorf("WriteMergeData: %v", err)
	}
	mergeData, err := manager.ReadMergeData()
	if err != nil {
		t.Errorf("ReadMergeData: %v", err)
	}
	if got, ok := mergeData["cutoff"].(string); !ok || got != "42" {
		t.Errorf("ReadMergeData: %v", got)
	}
	err = manager.RemoveMergeData()
	if err != nil {
		t.Errorf("RemoveMergeData: %v", err)
	}
}

func TestIndexManagerMisc(t *testing.T) {
	manager := NewIndexManager("", nil)
	manager.SetFolder(NewRAMFolder(""))
	if got, err := manager.MakeSnapshotFilename(); !strings.Contains(got, "snapshot") || err != nil {
		t.Errorf("MakeSnapshotFilename: %s, %v", got, err)
	}
	snapshot := NewSnapshot()
	snapshot.AddEntry("seg_4")
	snapshot.AddEntry("seg_5")
	if got := manager.HighestSegNum(snapshot); got != 5 {
		t.Errorf("HighestSegNum: %d", got)
	}
}

func TestIndexManagerRecycle(t *testing.T) {
	index := createTestIndex("foo", "bar", "baz")
	manager := NewIndexManager("", nil)
	manager.SetFolder(index)
	indexer, _ := OpenIndexer(&OpenIndexerArgs{Index: index})
	searcher, _ := OpenIndexSearcher(index)
	reader := searcher.GetReader().(PolyReader)
	delWriter := indexer.GetSegWriter().GetDelWriter()
	segReaders, err := manager.Recycle(reader, delWriter, 0, true)
	if err != nil || len(segReaders) != 1 {
		t.Errorf("Recycle: (%d SegReaders) %v", len(segReaders), err)
	}
}

func TestTermInfoMisc(t *testing.T) {
	tinfo := NewTermInfo(1000)
	if got := tinfo.GetDocFreq(); got != 1000 {
		t.Errorf("GetDocFreq: %d", got)
	}
	tinfo.SetDocFreq(1001)
	if got := tinfo.GetDocFreq(); got != 1001 {
		t.Errorf("Set/GetDocFreq: %d", got)
	}
	tinfo.SetLexFilePos(1002)
	if got := tinfo.GetLexFilePos(); got != 1002 {
		t.Errorf("Set/GetLexFilePos: %d", got)
	}
	tinfo.SetPostFilePos(1003)
	if got := tinfo.GetPostFilePos(); got != 1003 {
		t.Errorf("Set/GetPostFilePos: %d", got)
	}
	tinfo.SetSkipFilePos(1002)
	if got := tinfo.GetSkipFilePos(); got != 1002 {
		t.Errorf("Set/GetSkipFilePos: %d", got)
	}
	other := NewTermInfo(42)
	other.Mimic(tinfo)
	if got := other.GetDocFreq(); got != tinfo.GetDocFreq() {
		t.Errorf("Mimic: (%d != %d)", got, tinfo.GetDocFreq())
	}
	other = tinfo.Clone().(TermInfo)
	if got := other.GetDocFreq(); got != tinfo.GetDocFreq() {
		t.Errorf("Clone: (%d != %d)", got, tinfo.GetDocFreq())
	}
	tinfo.Reset()
	if got := tinfo.GetDocFreq(); got != 0 {
		t.Errorf("Reset: expected 0, got %d", got)
	}
}

func TestBitVecDelDocsMisc(t *testing.T) {
	folder := NewRAMFolder("")
	out, _ := folder.OpenOut("bits")
	out.WriteU32(0xDEADBEEF)
	out.Close()
	bv := NewBitVecDelDocs(folder, "bits")
	if !bv.Get(31) {
		t.Errorf("Get returned false")
	}
}

func TestTermVectorMisc(t *testing.T) {

	positions := []int32{0, 3}
	startOffsets := []int32{0, 20}
	endOffsets := []int32{2, 22}
	tv := NewTermVector("content", "red yellow green red blue", positions, startOffsets, endOffsets)
	if got := tv.GetPositions(); !reflect.DeepEqual(got, positions) {
		t.Errorf("GetPositions: %v", got)
	}
	if got := tv.GetStartOffsets(); !reflect.DeepEqual(got, startOffsets) {
		t.Errorf("GetStartOffsets: %v", got)
	}
	if got := tv.GetEndOffsets(); !reflect.DeepEqual(got, endOffsets) {
		t.Errorf("GetEndOffsets: %v", got)
	}

	folder := NewRAMFolder("")
	out, _ := folder.OpenOut("dump")
	tv.Serialize(out)
	out.Close()
	in, _ := folder.OpenIn("dump")
	dupe := clownfish.GetClass(tv).MakeObj().(TermVector).Deserialize(in)
	if !tv.Equals(dupe) {
		t.Errorf("Unsuccessful serialization round trip")
	}
}

func TestDocVectorMisc(t *testing.T) {
	schema := NewSchema()
	spec := NewFullTextType(NewStandardTokenizer())
	spec.SetHighlightable(true)
	schema.SpecField("content", spec)
	folder := NewRAMFolder("")
	indexer, _ := OpenIndexer(&OpenIndexerArgs{Index: folder, Schema: schema, Create: true})
	indexer.AddDoc(&testDoc{Content: "foo bar baz"})
	indexer.Commit()
	searcher, _ := OpenIndexSearcher(folder)
	dv := searcher.FetchDocVec(1)
	fieldBuf := dv.FieldBuf("content");
	if fieldBuf == nil {
		t.Errorf("FieldBuf returned nil")
	}
	dv.AddFieldBuf("content", fieldBuf)
	if got := dv.TermVector("content", "bar"); got == nil {
		t.Errorf("TermVector returned nil")
	}

	out, _ := folder.OpenOut("dump")
	dv.Serialize(out)
	out.Close()
	in, _ := folder.OpenIn("dump")
	dupe := clownfish.GetClass(dv).MakeObj().(DocVector).Deserialize(in)
	in.Close()
	if _, ok := dupe.(DocVector); !ok {
		t.Errorf("Serialize/Deserialize")
	}
}

func TestSnapshotMisc(t *testing.T) {
	var err error
	snapshot := NewSnapshot()
	snapshot.AddEntry("foo")
	snapshot.AddEntry("bar")
	snapshot.DeleteEntry("bar")
	if got := snapshot.NumEntries(); got != 1 {
		t.Errorf("Add/DeleteEntry, NumEntries: %d", got)
	}
	if got := snapshot.List(); !reflect.DeepEqual(got, []string{"foo"}) {
		t.Errorf("List: %v", got)
	}
	folder := NewRAMFolder("")
	err = snapshot.WriteFile(folder, "")
	if err != nil {
		t.Errorf("WriteFile: %v", err)
	}
	other := NewSnapshot()
	_, err = other.ReadFile(folder, "")
	if err != nil {
		t.Errorf("ReadFile: %v", err)
	}

	path := "snapshot_4.json"
	snapshot.SetPath(path)
	if got := snapshot.GetPath(); got != path {
		t.Errorf("SetPath/GetPath: %v", path)
	}
}
