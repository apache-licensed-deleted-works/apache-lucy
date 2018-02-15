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

package main

import "flag"
import "fmt"
import "io"
import "io/ioutil"
import "log"
import "os"
import "os/exec"
import "path"
import "runtime"

import "git-wip-us.apache.org/repos/asf/lucy-clownfish.git/compiler/go/cfc"

var packageName string = "git-wip-us.apache.org/repos/asf/lucy.git/go/lucy"
var cfPackageName string = "git-wip-us.apache.org/repos/asf/lucy-clownfish.git/runtime/go/clownfish"
var charmonizerC string = "../common/charmonizer.c"
var charmonizerEXE string = "charmonizer"
var charmonyH string = "charmony.h"
var buildDir string
var hostSrcDir string
var buildGO string
var configGO string
var cfbindGO string
var installedLibPath string

func init() {
	_, buildGO, _, _ = runtime.Caller(1)
	if buildGO == "<autogenerated>" {
		_, buildGO, _, _ = runtime.Caller(0)
	}
	buildDir = path.Dir(buildGO)
	hostSrcDir = path.Join(buildDir, "../c/src")
	configGO = path.Join(buildDir, "lucy", "config.go")
	cfbindGO = path.Join(buildDir, "lucy", "cfbind.go")
	var err error
	installedLibPath, err = cfc.InstalledLibPath(packageName)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	os.Chdir(buildDir)
	flag.Parse()
	action := "build"
	args := flag.Args()
	if len(args) > 0 {
		action = args[0]
	}
	switch action {
	case "build":
		build()
	case "clean":
		clean()
	case "test":
		test()
	case "install":
		install()
	default:
		log.Fatalf("Unrecognized action specified: %s", action)
	}
}

func current(orig, dest string) bool {

	destInfo, err := os.Stat(dest)
	if err != nil {
		if os.IsNotExist(err) {
			// If dest doesn't exist, we're not current.
			return false
		} else {
			log.Fatalf("Unexpected stat err: %s", err)
		}
	}

	// If source is newer than dest, we're not current.
	origInfo, err := os.Stat(orig)
	if err != nil {
		log.Fatalf("Unexpected: %s", err)
	}
	return origInfo.ModTime().Before(destInfo.ModTime())
}

func runCommand(name string, args ...string) {
	command := exec.Command(name, args...)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	err := command.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func configure() {
	if !current(charmonizerC, charmonizerEXE) {
		runCommand("cc", "-o", charmonizerEXE, charmonizerC)
	}
	if !current(charmonizerEXE, charmonyH) {
		runCommand("./charmonizer", "--cc=cc", "--enable-c", "--enable-go",
			"--enable-makefile", "--host=go", "--", "-std=gnu99", "-O2")
	}
}

func runCFC() {
	hierarchy := cfc.NewHierarchy("autogen")
	hierarchy.AddSourceDir("../core")
	hierarchy.AddSourceDir("../test")
	hierarchy.Build()
	autogenHeader := "Auto-generated by build.go.\n"
	coreBinding := cfc.NewBindCore(hierarchy, autogenHeader, "")
	modified := coreBinding.WriteAllModified(false)
	if modified {
		cfc.RegisterParcelPackage("Clownfish", cfPackageName)
		goBinding := cfc.NewBindGo(hierarchy)
		goBinding.SetHeader(autogenHeader)
		goBinding.SetSuppressInit(true)
		parcel := cfc.FetchParcel("Lucy")
		specClasses(parcel)
		packageDir := path.Join(buildDir, "lucy")
		goBinding.WriteBindings(parcel, packageDir)
		hierarchy.WriteLog()
	}
}

func specClasses(parcel *cfc.Parcel) {
	simpleBinding := cfc.NewGoClass(parcel, "Lucy::Simple")
	simpleBinding.SpecMethod("Add_Doc", "AddDoc(doc interface{}) error")
	simpleBinding.SpecMethod("Search", "Search(string, int, int, SortSpec) (int, error)")
	simpleBinding.SpecMethod("Next", "Next(hit interface{}) bool")
	simpleBinding.SpecMethod("", "Error() error")
	simpleBinding.SetSuppressCtor(true)
	simpleBinding.SetSuppressStruct(true)
	simpleBinding.Register()

	tokenBinding := cfc.NewGoClass(parcel, "Lucy::Analysis::Token")
	tokenBinding.SpecMethod("", "SetText(string)")
	tokenBinding.SpecMethod("", "GetText() string")
	tokenBinding.Register()

	analyzerBinding := cfc.NewGoClass(parcel, "Lucy::Analysis::Analyzer")
	analyzerBinding.SpecMethod("Split", "Split(string) []string")
	analyzerBinding.Register()

	polyAnalyzerBinding := cfc.NewGoClass(parcel, "Lucy::Analysis::PolyAnalyzer")
	polyAnalyzerBinding.SpecMethod("Get_Analyzers", "GetAnalyzers() []Analyzer")
	polyAnalyzerBinding.SetSuppressCtor(true)
	polyAnalyzerBinding.Register()

	docBinding := cfc.NewGoClass(parcel, "Lucy::Document::Doc")
	docBinding.SpecMethod("", "GetFields() map[string]interface{}")
	docBinding.SpecMethod("", "SetFields(map[string]interface{})")
	docBinding.SpecMethod("Field_Names", "FieldNames() []string")
	docBinding.Register()

	heatMapBinding := cfc.NewGoClass(parcel, "Lucy::Highlight::HeatMap")
	heatMapBinding.SetSuppressCtor(true)
	heatMapBinding.SpecMethod("Flatten_Spans", "flattenSpans([]Span) []Span")
	heatMapBinding.SpecMethod("Generate_Proximity_Boosts",
		"generateProximityBoosts([]Span) []Span")
	heatMapBinding.SpecMethod("Get_Spans", "getSpans() []Span")
	heatMapBinding.Register()

	indexerBinding := cfc.NewGoClass(parcel, "Lucy::Index::Indexer")
	indexerBinding.SpecMethod("", "Close() error")
	indexerBinding.SpecMethod("Add_Doc", "AddDoc(doc interface{}) error")
	indexerBinding.SpecMethod("Add_Index", "AddIndex(interface{}) error")
	indexerBinding.SpecMethod("Delete_By_Term", "DeleteByTerm(string, interface{}) error")
	indexerBinding.SpecMethod("Delete_By_Query", "DeleteByQuery(Query) error")
	indexerBinding.SpecMethod("Delete_By_Doc_ID", "DeleteByDocID(int32) error")
	indexerBinding.SpecMethod("Prepare_Commit", "PrepareCommit() error")
	indexerBinding.SpecMethod("Commit", "Commit() error")
	indexerBinding.SetSuppressStruct(true)
	indexerBinding.Register()

	dataReaderBinding := cfc.NewGoClass(parcel, "Lucy::Index::DataReader")
	dataReaderBinding.SpecMethod("Aggregator", "Aggregator([]DataReader, []int32) (DataReader, error)")
	dataReaderBinding.SpecMethod("Get_Segments", "GetSegments() []Segment")
	dataReaderBinding.SpecMethod("Close", "Close() error")
	dataReaderBinding.Register()

	ixReaderBinding := cfc.NewGoClass(parcel, "Lucy::Index::IndexReader")
	ixReaderBinding.SpecMethod("Seg_Readers", "SegReaders() []SegReader")
	ixReaderBinding.SpecMethod("Offsets", "Offsets() []int32")
	ixReaderBinding.SpecMethod("Obtain", "Obtain(string) (DataReader, error)")
	ixReaderBinding.Register()

	polyReaderBinding := cfc.NewGoClass(parcel, "Lucy::Index::PolyReader")
	polyReaderBinding.SetSuppressCtor(true)
	polyReaderBinding.Register()

	segReaderBinding := cfc.NewGoClass(parcel, "Lucy::Index::SegReader")
	segReaderBinding.SetSuppressCtor(true)
	segReaderBinding.Register()

	docReaderBinding := cfc.NewGoClass(parcel, "Lucy::Index::DocReader")
	docReaderBinding.SpecMethod("", "ReadDoc(int32, interface{}) error")
	docReaderBinding.SpecMethod("Fetch_Doc", "FetchDoc(int32) (HitDoc, error)")
	docReaderBinding.Register()

	hlReaderBinding := cfc.NewGoClass(parcel, "Lucy::Index::HighlightReader")
	hlReaderBinding.SpecMethod("Fetch_Doc_Vec", "FetchDocVec(int32) (DocVector, error)")
	hlReaderBinding.Register()

	sortReaderBinding := cfc.NewGoClass(parcel, "Lucy::Index::SortReader")
	sortReaderBinding.SpecMethod("Fetch_Sort_Cache", "fetchSortCache(string) (SortCache, error)")
	sortReaderBinding.Register()

	lexReaderBinding := cfc.NewGoClass(parcel, "Lucy::Index::LexiconReader")
	lexReaderBinding.SpecMethod("Lexicon", "Lexicon(string, interface{}) (Lexicon, error)")
	lexReaderBinding.SpecMethod("Doc_Freq", "DocFreq(string, interface{}) (uint32, error)")
	lexReaderBinding.SpecMethod("Fetch_Term_Info", "fetchTermInfo(string, interface{}) (TermInfo, error)")
	lexReaderBinding.Register()

	pListReaderBinding := cfc.NewGoClass(parcel, "Lucy::Index::PostingListReader")
	pListReaderBinding.SpecMethod("Posting_List", "PostingList(string, interface{}) (PostingList, error)")
	pListReaderBinding.Register()

	dwBinding := cfc.NewGoClass(parcel, "Lucy::Index::DataWriter")
	dwBinding.SpecMethod("Add_Inverted_Doc", "addInvertedDoc(Inverter, int32) error")
	dwBinding.SpecMethod("Add_Segment", "AddSegment(SegReader, []int32) error")
	dwBinding.SpecMethod("Delete_Segment", "DeleteSegment(SegReader) error")
	dwBinding.SpecMethod("Merge_Segment", "MergeSegment(SegReader, []int32) error")
	dwBinding.SpecMethod("Finish", "Finish() error")
	dwBinding.Register()

	segWriterBinding := cfc.NewGoClass(parcel, "Lucy::Index::SegWriter")
	segWriterBinding.SpecMethod("Prep_Seg_Dir", "PrepSegDir() error")
	segWriterBinding.SpecMethod("Add_Doc", "AddDoc(Doc, float32) error")
	segWriterBinding.Register()

	delWriterBinding := cfc.NewGoClass(parcel, "Lucy::Index::DeletionsWriter")
	delWriterBinding.SpecMethod("Delete_By_Term", "DeleteByTerm(string, interface{}) error")
	delWriterBinding.SpecMethod("Delete_By_Query", "DeleteByQuery(Query) error")
	delWriterBinding.SpecMethod("Delete_By_Doc_ID", "deleteByDocID(int32) error")
	delWriterBinding.SpecMethod("Generate_Doc_Map", "generateDocMap(Matcher, int32, int32) ([]int32, error)")
	delWriterBinding.SpecMethod("Seg_Deletions", "segDeletions(SegReader) (Matcher, error)")
	delWriterBinding.Register()

	bgMergerBinding := cfc.NewGoClass(parcel, "Lucy::Index::BackgroundMerger")
	bgMergerBinding.SpecMethod("Prepare_Commit", "PrepareCommit() error")
	bgMergerBinding.SpecMethod("Commit", "Commit() error")
	bgMergerBinding.Register()

	managerBinding := cfc.NewGoClass(parcel, "Lucy::Index::IndexManager")
	managerBinding.SpecMethod("Write_Merge_Data", "WriteMergeData(int64) error")
	managerBinding.SpecMethod("Read_Merge_Data", "ReadMergeData() (map[string]interface{}, error)")
	managerBinding.SpecMethod("Remove_Merge_Data", "RemoveMergeData() error")
	managerBinding.SpecMethod("Make_Snapshot_Filename", "MakeSnapshotFilename() (string, error)")
	managerBinding.SpecMethod("Recycle", "Recycle(PolyReader, DeletionsWriter, int64, bool) ([]SegReader, error)")
	managerBinding.Register()

	tvBinding := cfc.NewGoClass(parcel, "Lucy::Index::TermVector")
	tvBinding.SpecMethod("Get_Positions", "GetPositions() []int32")
	tvBinding.SpecMethod("Get_Start_Offsets", "GetStartOffsets() []int32")
	tvBinding.SpecMethod("Get_End_Offsets", "GetEndOffsets() []int32")
	tvBinding.SetSuppressCtor(true)
	tvBinding.Register()

	snapshotBinding := cfc.NewGoClass(parcel, "Lucy::Index::Snapshot")
	snapshotBinding.SpecMethod("List", "List() []string")
	snapshotBinding.SpecMethod("Read_File", "ReadFile(Folder, string) (Snapshot, error)")
	snapshotBinding.SpecMethod("Write_File", "WriteFile(Folder, string) error")
	snapshotBinding.Register()

	segBinding := cfc.NewGoClass(parcel, "Lucy::Index::Segment")
	segBinding.SpecMethod("Read_File", "ReadFile(Folder) error")
	segBinding.SpecMethod("Write_File", "WriteFile(Folder) error")
	segBinding.Register()

	sortCacheBinding := cfc.NewGoClass(parcel, "Lucy::Index::SortCache")
	sortCacheBinding.SpecMethod("Value", "Value(int32) (interface{}, error)")
	sortCacheBinding.SpecMethod("Ordinal", "Ordinal(int32) (int32, error)")
	sortCacheBinding.SpecMethod("Find", "Find(interface{}) (int32, error)")
	sortCacheBinding.Register()

	schemaBinding := cfc.NewGoClass(parcel, "Lucy::Plan::Schema")
	schemaBinding.SpecMethod("All_Fields", "AllFields() []string")
	schemaBinding.Register()

	searcherBinding := cfc.NewGoClass(parcel, "Lucy::Search::Searcher")
	searcherBinding.SpecMethod("Hits",
		"Hits(query interface{}, offset uint32, numWanted uint32, sortSpec SortSpec) (Hits, error)")
	searcherBinding.SpecMethod("Top_Docs", "topDocs(Query, uint32, SortSpec) (TopDocs, error)")
	searcherBinding.SpecMethod("Close", "Close() error")
	searcherBinding.SpecMethod("Fetch_Doc", "FetchDoc(int32) (HitDoc, error)")
	searcherBinding.SpecMethod("Fetch_Doc_Vec", "fetchDocVec(int32) (DocVector, error)")
	searcherBinding.SpecMethod("", "ReadDoc(int32, interface{}) error")
	searcherBinding.Register()

	qParserBinding := cfc.NewGoClass(parcel, "Lucy::Search::QueryParser")
	qParserBinding.SetSuppressCtor(true)
	qParserBinding.SpecMethod("Make_Phrase_Query", "MakePhraseQuery(string, []interface{}) PhraseQuery")
	qParserBinding.SpecMethod("Make_AND_Query", "MakeANDQuery([]Query) ANDQuery")
	qParserBinding.SpecMethod("Make_OR_Query", "MakeORQuery([]Query) ORQuery")
	qParserBinding.SpecMethod("Get_Fields", "getFields() []string")
	qParserBinding.Register()

	hitsBinding := cfc.NewGoClass(parcel, "Lucy::Search::Hits")
	hitsBinding.SpecMethod("Next", "Next(hit interface{}) bool")
	hitsBinding.SpecMethod("", "Error() error")
	hitsBinding.SetSuppressStruct(true)
	hitsBinding.Register()

	queryBinding := cfc.NewGoClass(parcel, "Lucy::Search::Query")
	queryBinding.SpecMethod("Make_Compiler", "MakeCompiler(Searcher, float32, bool) (Compiler, error)")
	queryBinding.Register()

	compilerBinding := cfc.NewGoClass(parcel, "Lucy::Search::Compiler")
	compilerBinding.SpecMethod("Make_Matcher", "MakeMatcher(SegReader, bool) (Matcher, error)")
	compilerBinding.Register()

	andQueryBinding := cfc.NewGoClass(parcel, "Lucy::Search::ANDQuery")
	andQueryBinding.SetSuppressCtor(true)
	andQueryBinding.Register()

	orQueryBinding := cfc.NewGoClass(parcel, "Lucy::Search::ORQuery")
	orQueryBinding.SetSuppressCtor(true)
	orQueryBinding.Register()

	matcherBinding := cfc.NewGoClass(parcel, "Lucy::Search::Matcher")
	matcherBinding.SpecMethod("Next", "Next() int32")
	matcherBinding.SpecMethod("", "Error() error")
	matcherBinding.SetSuppressStruct(true)
	matcherBinding.Register()

	andMatcherBinding := cfc.NewGoClass(parcel, "Lucy::Search::ANDMatcher")
	andMatcherBinding.SetSuppressCtor(true)
	andMatcherBinding.Register()

	orMatcherBinding := cfc.NewGoClass(parcel, "Lucy::Search::ORMatcher")
	orMatcherBinding.SetSuppressCtor(true)
	orMatcherBinding.Register()

	orScorerBinding := cfc.NewGoClass(parcel, "Lucy::Search::ORScorer")
	orScorerBinding.SetSuppressCtor(true)
	orScorerBinding.Register()

	seriesMatcherBinding := cfc.NewGoClass(parcel, "Lucy::Search::SeriesMatcher")
	seriesMatcherBinding.SetSuppressCtor(true)
	seriesMatcherBinding.Register()

	bitVecBinding := cfc.NewGoClass(parcel, "Lucy::Object::BitVector")
	bitVecBinding.SpecMethod("To_Array", "ToArray() []bool")
	bitVecBinding.Register()

	mockMatcherBinding := cfc.NewGoClass(parcel, "LucyX::Search::MockMatcher")
	mockMatcherBinding.SetSuppressCtor(true)
	mockMatcherBinding.Register()

	topDocsBinding := cfc.NewGoClass(parcel, "Lucy::Search::TopDocs")
	topDocsBinding.SetSuppressCtor(true)
	topDocsBinding.SpecMethod("Set_Match_Docs", "SetMatchDocs([]MatchDoc)")
	topDocsBinding.SpecMethod("Get_Match_Docs", "GetMatchDocs() []MatchDoc")
	topDocsBinding.Register()

	sortSpecBinding := cfc.NewGoClass(parcel, "Lucy::Search::SortSpec")
	sortSpecBinding.SetSuppressCtor(true)
	sortSpecBinding.SpecMethod("Get_Rules", "GetRules() []SortRule")
	sortSpecBinding.Register()

	sortCollBinding := cfc.NewGoClass(parcel, "Lucy::Search::Collector::SortCollector")
	sortCollBinding.SpecMethod("Pop_Match_Docs", "PopMatchDocs() []MatchDoc")
	sortCollBinding.Register()

	inStreamBinding := cfc.NewGoClass(parcel, "Lucy::Store::InStream")
	inStreamBinding.SpecMethod("Reopen", "Reopen(string, int64, int64) (InStream, error)")
	inStreamBinding.SpecMethod("Close", "Close() error")
	inStreamBinding.SpecMethod("Seek", "Seek(int64) error")
	inStreamBinding.SpecMethod("", "ReadBytes([]byte, int) error")
	inStreamBinding.SpecMethod("", "ReadString() (string, error)")
	inStreamBinding.SpecMethod("Read_I8", "ReadI8() (int8, error)")
	inStreamBinding.SpecMethod("Read_I32", "ReadI32() (int32, error)")
	inStreamBinding.SpecMethod("Read_I64", "ReadI64() (int64, error)")
	inStreamBinding.SpecMethod("Read_U8", "ReadU8() (uint8, error)")
	inStreamBinding.SpecMethod("Read_U32", "ReadU32() (uint32, error)")
	inStreamBinding.SpecMethod("Read_U64", "ReadU64() (uint64, error)")
	inStreamBinding.SpecMethod("Read_CI32", "ReadCI32() (int32, error)")
	inStreamBinding.SpecMethod("Read_CU32", "ReadCU32() (uint32, error)")
	inStreamBinding.SpecMethod("Read_CI64", "ReadCI64() (int64, error)")
	inStreamBinding.SpecMethod("Read_CU64", "ReadCU64() (uint64, error)")
	inStreamBinding.SpecMethod("Read_F32", "ReadF32() (float32, error)")
	inStreamBinding.SpecMethod("Read_F64", "ReadF64() (float64, error)")
	inStreamBinding.Register()

	outStreamBinding := cfc.NewGoClass(parcel, "Lucy::Store::OutStream")
	outStreamBinding.SpecMethod("Close", "Close() error")
	outStreamBinding.SpecMethod("Grow", "Grow(int64) error")
	outStreamBinding.SpecMethod("Align", "Align(int64) error")
	outStreamBinding.SpecMethod("", "WriteBytes([]byte, int) error")
	outStreamBinding.SpecMethod("", "WriteString(string) error")
	outStreamBinding.SpecMethod("Write_I8", "WriteI8(int8) error")
	outStreamBinding.SpecMethod("Write_I32", "WriteI32(int32) error")
	outStreamBinding.SpecMethod("Write_I64", "WriteI64(int64) error")
	outStreamBinding.SpecMethod("Write_U8", "WriteU8(uint8) error")
	outStreamBinding.SpecMethod("Write_U32", "WriteU32(uint32) error")
	outStreamBinding.SpecMethod("Write_U64", "WriteU64(uint64) error")
	outStreamBinding.SpecMethod("Write_CI32", "WriteCI32(int32) error")
	outStreamBinding.SpecMethod("Write_CU32", "WriteCU32(uint32) error")
	outStreamBinding.SpecMethod("Write_CI64", "WriteCI64(int64) error")
	outStreamBinding.SpecMethod("Write_CU64", "WriteCU64(uint64) error")
	outStreamBinding.SpecMethod("Write_F32", "WriteF32(float32) error")
	outStreamBinding.SpecMethod("Write_F64", "WriteF64(float64) error")
	outStreamBinding.SpecMethod("Absorb", "Absorb(InStream) error")
	outStreamBinding.Register()

	folderBinding := cfc.NewGoClass(parcel, "Lucy::Store::Folder")
	folderBinding.SpecMethod("Initialize", "Initialize() error")
	folderBinding.SpecMethod("Open_Out", "OpenOut(string) (OutStream, error)")
	folderBinding.SpecMethod("Open_In", "OpenIn(string) (InStream, error)")
	folderBinding.SpecMethod("Open_FileHandle", "OpenFileHandle(string, uint32) (FileHandle, error)")
	folderBinding.SpecMethod("Open_Dir", "OpenDir(string) (DirHandle, error)")
	folderBinding.SpecMethod("MkDir", "MkDir(string) error")
	folderBinding.SpecMethod("List", "List(string) ([]string, error)")
	folderBinding.SpecMethod("List_R", "ListR(string) ([]string, error)")
	folderBinding.SpecMethod("Rename", "Rename(string, string) error")
	folderBinding.SpecMethod("Hard_Link", "HardLink(string, string) error")
	folderBinding.SpecMethod("Slurp_File", "SlurpFile(string) ([]byte, error)")
	folderBinding.SpecMethod("Consolidate", "Consolidate(string) error")
	folderBinding.SpecMethod("Local_Open_In", "LocalOpenIn(string) (InStream, error)")
	folderBinding.SpecMethod("Local_Open_FileHandle", "LocalOpenFileHandle(string, uint32) (FileHandle, error)")
	folderBinding.SpecMethod("Local_Open_Dir", "LocalOpenDir() (DirHandle, error)")
	folderBinding.SpecMethod("Local_MkDir", "LocalMkDir(string) error")
	folderBinding.Register()

	fhBinding := cfc.NewGoClass(parcel, "Lucy::Store::FileHandle")
	fhBinding.SpecMethod("", "Write([]byte, int) error")
	fhBinding.SpecMethod("", "Read([]byte, int64, int) error")
	fhBinding.SpecMethod("Window", "Window(FileWindow, int64, int64) error")
	fhBinding.SpecMethod("Release_Window", "ReleaseWindow(FileWindow) error")
	fhBinding.SpecMethod("Grow", "Grow(int64) error")
	fhBinding.SpecMethod("Close", "Close() error")
	fhBinding.Register()

	dhBinding := cfc.NewGoClass(parcel, "Lucy::Store::DirHandle")
	dhBinding.SpecMethod("Close", "Close() error")
	dhBinding.SpecMethod("Next", "next() bool")
	dhBinding.SpecMethod("", "Error() error")
	dhBinding.SetSuppressStruct(true)
	dhBinding.Register()

	lockBinding := cfc.NewGoClass(parcel, "Lucy::Store::Lock")
	lockBinding.SpecMethod("Request", "Request() error")
	lockBinding.SpecMethod("Obtain", "Obtain() error")
	lockBinding.SpecMethod("Release", "Release() error")
	lockBinding.SpecMethod("Clear_Stale", "ClearStale() error")
	lockBinding.Register()

	cfWriterBinding := cfc.NewGoClass(parcel, "Lucy::Store::CompoundFileWriter")
	cfWriterBinding.SpecMethod("Consolidate", "Consolidate() error")
	cfWriterBinding.Register()

	stepperBinding := cfc.NewGoClass(parcel, "Lucy::Util::Stepper")
	stepperBinding.SpecMethod("Write_Key_Frame", "WriteKeyFrame(OutStream, interface{}) error")
	stepperBinding.SpecMethod("Write_Delta", "WriteDelta(OutStream, interface{}) error")
	stepperBinding.SpecMethod("Read_Key_Frame", "ReadKeyFrame(InStream) error")
	stepperBinding.SpecMethod("Read_Delta", "ReadDelta(InStream) error")
	stepperBinding.SpecMethod("Read_Record", "readRecord(InStream) error")
	stepperBinding.Register()
}

func build() {
	configure()
	runCFC()
	runCommand("make", "-j", "static")
	writeConfigGO()
	runCommand("go", "build", packageName)
}

func test() {
	build()
	runCommand("go", "test", packageName)
}

func copyFile(source, dest string) {
	sourceFH, err := os.Open(source)
	if err != nil {
		log.Fatal(err)
	}
	defer sourceFH.Close()
	destFH, err := os.Create(dest)
	if err != nil {
		log.Fatal(err)
	}
	defer destFH.Close()
	_, err = io.Copy(destFH, sourceFH)
	if err != nil {
		log.Fatalf("io.Copy from %s to %s failed: %s", source, dest, err)
	}
}

func installStaticLib() {
	tempLibPath := path.Join(buildDir, "liblucy.a")
	destDir := path.Dir(installedLibPath)
	if _, err := os.Stat(destDir); os.IsNotExist(err) {
		err = os.MkdirAll(destDir, 0755)
		if err != nil {
			log.Fatalf("Can't create dir '%s': %s", destDir, err)
		}
	}
	os.Remove(installedLibPath)
	copyFile(tempLibPath, installedLibPath)
}

func install() {
	build()
	runCommand("go", "install", packageName)
	installStaticLib()
}

func writeConfigGO() {
	if current(buildGO, configGO) {
		return
	}
	installedLibDir := path.Dir(installedLibPath)
	cfLibPath, err := cfc.InstalledLibPath(cfPackageName)
	if err != nil {
		log.Fatal(err)
	}
	cfLibDir := path.Dir(cfLibPath)
	content := fmt.Sprintf(
		"// Auto-generated by build.go, specifying absolute path to static lib.\n"+
			"package lucy\n"+
			"// #cgo CFLAGS: -I%s/../core\n"+
			"// #cgo CFLAGS: -I%s\n"+
			"// #cgo CFLAGS: -I%s/autogen/include\n"+
			"// #cgo LDFLAGS: -L%s\n"+
			"// #cgo LDFLAGS: -L%s\n"+
			"// #cgo LDFLAGS: -L%s\n"+
			"// #cgo LDFLAGS: -ltestlucy\n"+
			"// #cgo LDFLAGS: -llucy\n"+
			"// #cgo LDFLAGS: -lclownfish\n"+
			"// #cgo LDFLAGS: -lm\n"+
			"import \"C\"\n",
		buildDir, buildDir, buildDir, buildDir, installedLibDir, cfLibDir)
	ioutil.WriteFile(configGO, []byte(content), 0666)
}

func clean() {
	fmt.Println("Cleaning")
	if _, err := os.Stat("Makefile"); !os.IsNotExist(err) {
		runCommand("make", "clean")
	}
	files := []string{charmonizerEXE, "charmony.h", "Makefile", configGO, cfbindGO}
	for _, file := range files {
		err := os.Remove(file)
		if err == nil {
			fmt.Println("Removing", file)
		} else if !os.IsNotExist(err) {
			log.Fatal(err)
		}
	}
}
