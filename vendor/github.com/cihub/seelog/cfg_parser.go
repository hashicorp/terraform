// Copyright (c) 2012 - Cloud Instruments Co., Ltd.
//
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are met:
//
// 1. Redistributions of source code must retain the above copyright notice, this
//    list of conditions and the following disclaimer.
// 2. Redistributions in binary form must reproduce the above copyright notice,
//    this list of conditions and the following disclaimer in the documentation
//    and/or other materials provided with the distribution.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE LIABLE FOR
// ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package seelog

import (
	"crypto/tls"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

// Names of elements of seelog config.
const (
	seelogConfigID                   = "seelog"
	outputsID                        = "outputs"
	formatsID                        = "formats"
	minLevelID                       = "minlevel"
	maxLevelID                       = "maxlevel"
	levelsID                         = "levels"
	exceptionsID                     = "exceptions"
	exceptionID                      = "exception"
	funcPatternID                    = "funcpattern"
	filePatternID                    = "filepattern"
	formatID                         = "format"
	formatAttrID                     = "format"
	formatKeyAttrID                  = "id"
	outputFormatID                   = "formatid"
	pathID                           = "path"
	fileWriterID                     = "file"
	smtpWriterID                     = "smtp"
	senderaddressID                  = "senderaddress"
	senderNameID                     = "sendername"
	recipientID                      = "recipient"
	mailHeaderID                     = "header"
	mailHeaderNameID                 = "name"
	mailHeaderValueID                = "value"
	addressID                        = "address"
	hostNameID                       = "hostname"
	hostPortID                       = "hostport"
	userNameID                       = "username"
	userPassID                       = "password"
	cACertDirpathID                  = "cacertdirpath"
	subjectID                        = "subject"
	splitterDispatcherID             = "splitter"
	consoleWriterID                  = "console"
	customReceiverID                 = "custom"
	customNameAttrID                 = "name"
	customNameDataAttrPrefix         = "data-"
	filterDispatcherID               = "filter"
	filterLevelsAttrID               = "levels"
	rollingfileWriterID              = "rollingfile"
	rollingFileTypeAttr              = "type"
	rollingFilePathAttr              = "filename"
	rollingFileMaxSizeAttr           = "maxsize"
	rollingFileMaxRollsAttr          = "maxrolls"
	rollingFileNameModeAttr          = "namemode"
	rollingFileDataPatternAttr       = "datepattern"
	rollingFileArchiveAttr           = "archivetype"
	rollingFileArchivePathAttr       = "archivepath"
	rollingFileArchiveExplodedAttr   = "archiveexploded"
	rollingFileFullNameAttr          = "fullname"
	bufferedWriterID                 = "buffered"
	bufferedSizeAttr                 = "size"
	bufferedFlushPeriodAttr          = "flushperiod"
	loggerTypeFromStringAttr         = "type"
	asyncLoggerIntervalAttr          = "asyncinterval"
	adaptLoggerMinIntervalAttr       = "mininterval"
	adaptLoggerMaxIntervalAttr       = "maxinterval"
	adaptLoggerCriticalMsgCountAttr  = "critmsgcount"
	predefinedPrefix                 = "std:"
	connWriterID                     = "conn"
	connWriterAddrAttr               = "addr"
	connWriterNetAttr                = "net"
	connWriterReconnectOnMsgAttr     = "reconnectonmsg"
	connWriterUseTLSAttr             = "tls"
	connWriterInsecureSkipVerifyAttr = "insecureskipverify"
)

// CustomReceiverProducer is the signature of the function CfgParseParams needs to create
// custom receivers.
type CustomReceiverProducer func(CustomReceiverInitArgs) (CustomReceiver, error)

// CfgParseParams represent specific parse options or flags used by parser. It is used if seelog parser needs
// some special directives or additional info to correctly parse a config.
type CfgParseParams struct {
	// CustomReceiverProducers expose the same functionality as RegisterReceiver func
	// but only in the scope (context) of the config parse func instead of a global package scope.
	//
	// It means that if you use custom receivers in your code, you may either register them globally once with
	// RegisterReceiver or you may call funcs like LoggerFromParamConfigAsFile (with 'ParamConfig')
	// and use CustomReceiverProducers to provide custom producer funcs.
	//
	// A producer func is called when config parser processes a '<custom>' element. It takes the 'name' attribute
	// of the element and tries to find a match in two places:
	// 1) CfgParseParams.CustomReceiverProducers map
	// 2) Global type map, filled by RegisterReceiver
	//
	// If a match is found in the CustomReceiverProducers map, parser calls the corresponding producer func
	// passing the init args to it.	The func takes exactly the same args as CustomReceiver.AfterParse.
	// The producer func must return a correct receiver or an error. If case of error, seelog will behave
	// in the same way as with any other config error.
	//
	// You may use this param to set custom producers in case you need to pass some context when instantiating
	// a custom receiver or if you frequently change custom receivers with different parameters or in any other
	// situation where package-level registering (RegisterReceiver) is not an option for you.
	CustomReceiverProducers map[string]CustomReceiverProducer
}

func (cfg *CfgParseParams) String() string {
	return fmt.Sprintf("CfgParams: {custom_recs=%d}", len(cfg.CustomReceiverProducers))
}

type elementMapEntry struct {
	constructor func(node *xmlNode, formatFromParent *formatter, formats map[string]*formatter, cfg *CfgParseParams) (interface{}, error)
}

var elementMap map[string]elementMapEntry
var predefinedFormats map[string]*formatter

func init() {
	elementMap = map[string]elementMapEntry{
		fileWriterID:         {createfileWriter},
		splitterDispatcherID: {createSplitter},
		customReceiverID:     {createCustomReceiver},
		filterDispatcherID:   {createFilter},
		consoleWriterID:      {createConsoleWriter},
		rollingfileWriterID:  {createRollingFileWriter},
		bufferedWriterID:     {createbufferedWriter},
		smtpWriterID:         {createSMTPWriter},
		connWriterID:         {createconnWriter},
	}

	err := fillPredefinedFormats()
	if err != nil {
		panic(fmt.Sprintf("Seelog couldn't start: predefined formats creation failed. Error: %s", err.Error()))
	}
}

func fillPredefinedFormats() error {
	predefinedFormatsWithoutPrefix := map[string]string{
		"xml-debug":       `<time>%Ns</time><lev>%Lev</lev><msg>%Msg</msg><path>%RelFile</path><func>%Func</func><line>%Line</line>`,
		"xml-debug-short": `<t>%Ns</t><l>%l</l><m>%Msg</m><p>%RelFile</p><f>%Func</f>`,
		"xml":             `<time>%Ns</time><lev>%Lev</lev><msg>%Msg</msg>`,
		"xml-short":       `<t>%Ns</t><l>%l</l><m>%Msg</m>`,

		"json-debug":       `{"time":%Ns,"lev":"%Lev","msg":"%Msg","path":"%RelFile","func":"%Func","line":"%Line"}`,
		"json-debug-short": `{"t":%Ns,"l":"%Lev","m":"%Msg","p":"%RelFile","f":"%Func"}`,
		"json":             `{"time":%Ns,"lev":"%Lev","msg":"%Msg"}`,
		"json-short":       `{"t":%Ns,"l":"%Lev","m":"%Msg"}`,

		"debug":       `[%LEVEL] %RelFile:%Func.%Line %Date %Time %Msg%n`,
		"debug-short": `[%LEVEL] %Date %Time %Msg%n`,
		"fast":        `%Ns %l %Msg%n`,
	}

	predefinedFormats = make(map[string]*formatter)

	for formatKey, format := range predefinedFormatsWithoutPrefix {
		formatter, err := NewFormatter(format)
		if err != nil {
			return err
		}

		predefinedFormats[predefinedPrefix+formatKey] = formatter
	}

	return nil
}

// configFromXMLDecoder parses data from a given XML decoder.
// Returns parsed config which can be used to create logger in case no errors occured.
// Returns error if format is incorrect or anything happened.
func configFromXMLDecoder(xmlParser *xml.Decoder, rootNode xml.Token) (*configForParsing, error) {
	return configFromXMLDecoderWithConfig(xmlParser, rootNode, nil)
}

// configFromXMLDecoderWithConfig parses data from a given XML decoder.
// Returns parsed config which can be used to create logger in case no errors occured.
// Returns error if format is incorrect or anything happened.
func configFromXMLDecoderWithConfig(xmlParser *xml.Decoder, rootNode xml.Token, cfg *CfgParseParams) (*configForParsing, error) {
	_, ok := rootNode.(xml.StartElement)
	if !ok {
		return nil, errors.New("rootNode must be XML startElement")
	}

	config, err := unmarshalNode(xmlParser, rootNode)
	if err != nil {
		return nil, err
	}
	if config == nil {
		return nil, errors.New("xml has no content")
	}

	return configFromXMLNodeWithConfig(config, cfg)
}

// configFromReader parses data from a given reader.
// Returns parsed config which can be used to create logger in case no errors occured.
// Returns error if format is incorrect or anything happened.
func configFromReader(reader io.Reader) (*configForParsing, error) {
	return configFromReaderWithConfig(reader, nil)
}

// configFromReaderWithConfig parses data from a given reader.
// Returns parsed config which can be used to create logger in case no errors occured.
// Returns error if format is incorrect or anything happened.
func configFromReaderWithConfig(reader io.Reader, cfg *CfgParseParams) (*configForParsing, error) {
	config, err := unmarshalConfig(reader)
	if err != nil {
		return nil, err
	}

	if config.name != seelogConfigID {
		return nil, errors.New("root xml tag must be '" + seelogConfigID + "'")
	}

	return configFromXMLNodeWithConfig(config, cfg)
}

func configFromXMLNodeWithConfig(config *xmlNode, cfg *CfgParseParams) (*configForParsing, error) {
	err := checkUnexpectedAttribute(
		config,
		minLevelID,
		maxLevelID,
		levelsID,
		loggerTypeFromStringAttr,
		asyncLoggerIntervalAttr,
		adaptLoggerMinIntervalAttr,
		adaptLoggerMaxIntervalAttr,
		adaptLoggerCriticalMsgCountAttr,
	)
	if err != nil {
		return nil, err
	}

	err = checkExpectedElements(config, optionalElement(outputsID), optionalElement(formatsID), optionalElement(exceptionsID))
	if err != nil {
		return nil, err
	}

	constraints, err := getConstraints(config)
	if err != nil {
		return nil, err
	}

	exceptions, err := getExceptions(config)
	if err != nil {
		return nil, err
	}
	err = checkDistinctExceptions(exceptions)
	if err != nil {
		return nil, err
	}

	formats, err := getFormats(config)
	if err != nil {
		return nil, err
	}

	dispatcher, err := getOutputsTree(config, formats, cfg)
	if err != nil {
		// If we open several files, but then fail to parse the config, we should close
		// those files before reporting that config is invalid.
		if dispatcher != nil {
			dispatcher.Close()
		}

		return nil, err
	}

	loggerType, logData, err := getloggerTypeFromStringData(config)
	if err != nil {
		return nil, err
	}

	return newFullLoggerConfig(constraints, exceptions, dispatcher, loggerType, logData, cfg)
}

func getConstraints(node *xmlNode) (logLevelConstraints, error) {
	minLevelStr, isMinLevel := node.attributes[minLevelID]
	maxLevelStr, isMaxLevel := node.attributes[maxLevelID]
	levelsStr, isLevels := node.attributes[levelsID]

	if isLevels && (isMinLevel && isMaxLevel) {
		return nil, errors.New("for level declaration use '" + levelsID + "'' OR '" + minLevelID +
			"', '" + maxLevelID + "'")
	}

	offString := LogLevel(Off).String()

	if (isLevels && strings.TrimSpace(levelsStr) == offString) ||
		(isMinLevel && !isMaxLevel && minLevelStr == offString) {

		return NewOffConstraints()
	}

	if isLevels {
		levels, err := parseLevels(levelsStr)
		if err != nil {
			return nil, err
		}
		return NewListConstraints(levels)
	}

	var minLevel = LogLevel(TraceLvl)
	if isMinLevel {
		found := true
		minLevel, found = LogLevelFromString(minLevelStr)
		if !found {
			return nil, errors.New("declared " + minLevelID + " not found: " + minLevelStr)
		}
	}

	var maxLevel = LogLevel(CriticalLvl)
	if isMaxLevel {
		found := true
		maxLevel, found = LogLevelFromString(maxLevelStr)
		if !found {
			return nil, errors.New("declared " + maxLevelID + " not found: " + maxLevelStr)
		}
	}

	return NewMinMaxConstraints(minLevel, maxLevel)
}

func parseLevels(str string) ([]LogLevel, error) {
	levelsStrArr := strings.Split(strings.Replace(str, " ", "", -1), ",")
	var levels []LogLevel
	for _, levelStr := range levelsStrArr {
		level, found := LogLevelFromString(levelStr)
		if !found {
			return nil, errors.New("declared level not found: " + levelStr)
		}

		levels = append(levels, level)
	}

	return levels, nil
}

func getExceptions(config *xmlNode) ([]*LogLevelException, error) {
	var exceptions []*LogLevelException

	var exceptionsNode *xmlNode
	for _, child := range config.children {
		if child.name == exceptionsID {
			exceptionsNode = child
			break
		}
	}

	if exceptionsNode == nil {
		return exceptions, nil
	}

	err := checkUnexpectedAttribute(exceptionsNode)
	if err != nil {
		return nil, err
	}

	err = checkExpectedElements(exceptionsNode, multipleMandatoryElements("exception"))
	if err != nil {
		return nil, err
	}

	for _, exceptionNode := range exceptionsNode.children {
		if exceptionNode.name != exceptionID {
			return nil, errors.New("incorrect nested element in exceptions section: " + exceptionNode.name)
		}

		err := checkUnexpectedAttribute(exceptionNode, minLevelID, maxLevelID, levelsID, funcPatternID, filePatternID)
		if err != nil {
			return nil, err
		}

		constraints, err := getConstraints(exceptionNode)
		if err != nil {
			return nil, errors.New("incorrect " + exceptionsID + " node: " + err.Error())
		}

		funcPattern, isFuncPattern := exceptionNode.attributes[funcPatternID]
		filePattern, isFilePattern := exceptionNode.attributes[filePatternID]
		if !isFuncPattern {
			funcPattern = "*"
		}
		if !isFilePattern {
			filePattern = "*"
		}

		exception, err := NewLogLevelException(funcPattern, filePattern, constraints)
		if err != nil {
			return nil, errors.New("incorrect exception node: " + err.Error())
		}

		exceptions = append(exceptions, exception)
	}

	return exceptions, nil
}

func checkDistinctExceptions(exceptions []*LogLevelException) error {
	for i, exception := range exceptions {
		for j, exception1 := range exceptions {
			if i == j {
				continue
			}

			if exception.FuncPattern() == exception1.FuncPattern() &&
				exception.FilePattern() == exception1.FilePattern() {

				return fmt.Errorf("there are two or more duplicate exceptions. Func: %v, file %v",
					exception.FuncPattern(), exception.FilePattern())
			}
		}
	}

	return nil
}

func getFormats(config *xmlNode) (map[string]*formatter, error) {
	formats := make(map[string]*formatter, 0)

	var formatsNode *xmlNode
	for _, child := range config.children {
		if child.name == formatsID {
			formatsNode = child
			break
		}
	}

	if formatsNode == nil {
		return formats, nil
	}

	err := checkUnexpectedAttribute(formatsNode)
	if err != nil {
		return nil, err
	}

	err = checkExpectedElements(formatsNode, multipleMandatoryElements("format"))
	if err != nil {
		return nil, err
	}

	for _, formatNode := range formatsNode.children {
		if formatNode.name != formatID {
			return nil, errors.New("incorrect nested element in " + formatsID + " section: " + formatNode.name)
		}

		err := checkUnexpectedAttribute(formatNode, formatKeyAttrID, formatID)
		if err != nil {
			return nil, err
		}

		id, isID := formatNode.attributes[formatKeyAttrID]
		formatStr, isFormat := formatNode.attributes[formatAttrID]
		if !isID {
			return nil, errors.New("format has no '" + formatKeyAttrID + "' attribute")
		}
		if !isFormat {
			return nil, errors.New("format[" + id + "] has no '" + formatAttrID + "' attribute")
		}

		formatter, err := NewFormatter(formatStr)
		if err != nil {
			return nil, err
		}

		formats[id] = formatter
	}

	return formats, nil
}

func getloggerTypeFromStringData(config *xmlNode) (logType loggerTypeFromString, logData interface{}, err error) {
	logTypeStr, loggerTypeExists := config.attributes[loggerTypeFromStringAttr]

	if !loggerTypeExists {
		return defaultloggerTypeFromString, nil, nil
	}

	logType, found := getLoggerTypeFromString(logTypeStr)

	if !found {
		return 0, nil, fmt.Errorf("unknown logger type: %s", logTypeStr)
	}

	if logType == asyncTimerloggerTypeFromString {
		intervalStr, intervalExists := config.attributes[asyncLoggerIntervalAttr]
		if !intervalExists {
			return 0, nil, newMissingArgumentError(config.name, asyncLoggerIntervalAttr)
		}

		interval, err := strconv.ParseUint(intervalStr, 10, 32)
		if err != nil {
			return 0, nil, err
		}

		logData = asyncTimerLoggerData{uint32(interval)}
	} else if logType == adaptiveLoggerTypeFromString {

		// Min interval
		minIntStr, minIntExists := config.attributes[adaptLoggerMinIntervalAttr]
		if !minIntExists {
			return 0, nil, newMissingArgumentError(config.name, adaptLoggerMinIntervalAttr)
		}
		minInterval, err := strconv.ParseUint(minIntStr, 10, 32)
		if err != nil {
			return 0, nil, err
		}

		// Max interval
		maxIntStr, maxIntExists := config.attributes[adaptLoggerMaxIntervalAttr]
		if !maxIntExists {
			return 0, nil, newMissingArgumentError(config.name, adaptLoggerMaxIntervalAttr)
		}
		maxInterval, err := strconv.ParseUint(maxIntStr, 10, 32)
		if err != nil {
			return 0, nil, err
		}

		// Critical msg count
		criticalMsgCountStr, criticalMsgCountExists := config.attributes[adaptLoggerCriticalMsgCountAttr]
		if !criticalMsgCountExists {
			return 0, nil, newMissingArgumentError(config.name, adaptLoggerCriticalMsgCountAttr)
		}
		criticalMsgCount, err := strconv.ParseUint(criticalMsgCountStr, 10, 32)
		if err != nil {
			return 0, nil, err
		}

		logData = adaptiveLoggerData{uint32(minInterval), uint32(maxInterval), uint32(criticalMsgCount)}
	}

	return logType, logData, nil
}

func getOutputsTree(config *xmlNode, formats map[string]*formatter, cfg *CfgParseParams) (dispatcherInterface, error) {
	var outputsNode *xmlNode
	for _, child := range config.children {
		if child.name == outputsID {
			outputsNode = child
			break
		}
	}

	if outputsNode != nil {
		err := checkUnexpectedAttribute(outputsNode, outputFormatID)
		if err != nil {
			return nil, err
		}

		formatter, err := getCurrentFormat(outputsNode, DefaultFormatter, formats)
		if err != nil {
			return nil, err
		}

		output, err := createSplitter(outputsNode, formatter, formats, cfg)
		if err != nil {
			return nil, err
		}

		dispatcher, ok := output.(dispatcherInterface)
		if ok {
			return dispatcher, nil
		}
	}

	console, err := NewConsoleWriter()
	if err != nil {
		return nil, err
	}
	return NewSplitDispatcher(DefaultFormatter, []interface{}{console})
}

func getCurrentFormat(node *xmlNode, formatFromParent *formatter, formats map[string]*formatter) (*formatter, error) {
	formatID, isFormatID := node.attributes[outputFormatID]
	if !isFormatID {
		return formatFromParent, nil
	}

	format, ok := formats[formatID]
	if ok {
		return format, nil
	}

	// Test for predefined format match
	pdFormat, pdOk := predefinedFormats[formatID]

	if !pdOk {
		return nil, errors.New("formatid = '" + formatID + "' doesn't exist")
	}

	return pdFormat, nil
}

func createInnerReceivers(node *xmlNode, format *formatter, formats map[string]*formatter, cfg *CfgParseParams) ([]interface{}, error) {
	var outputs []interface{}
	for _, childNode := range node.children {
		entry, ok := elementMap[childNode.name]
		if !ok {
			return nil, errors.New("unnknown tag '" + childNode.name + "' in outputs section")
		}

		output, err := entry.constructor(childNode, format, formats, cfg)
		if err != nil {
			return nil, err
		}

		outputs = append(outputs, output)
	}

	return outputs, nil
}

func createSplitter(node *xmlNode, formatFromParent *formatter, formats map[string]*formatter, cfg *CfgParseParams) (interface{}, error) {
	err := checkUnexpectedAttribute(node, outputFormatID)
	if err != nil {
		return nil, err
	}

	if !node.hasChildren() {
		return nil, errNodeMustHaveChildren
	}

	currentFormat, err := getCurrentFormat(node, formatFromParent, formats)
	if err != nil {
		return nil, err
	}

	receivers, err := createInnerReceivers(node, currentFormat, formats, cfg)
	if err != nil {
		return nil, err
	}

	return NewSplitDispatcher(currentFormat, receivers)
}

func createCustomReceiver(node *xmlNode, formatFromParent *formatter, formats map[string]*formatter, cfg *CfgParseParams) (interface{}, error) {
	dataCustomPrefixes := make(map[string]string)
	// Expecting only 'formatid', 'name' and 'data-' attrs
	for attr, attrval := range node.attributes {
		isExpected := false
		if attr == outputFormatID ||
			attr == customNameAttrID {
			isExpected = true
		}
		if strings.HasPrefix(attr, customNameDataAttrPrefix) {
			dataCustomPrefixes[attr[len(customNameDataAttrPrefix):]] = attrval
			isExpected = true
		}
		if !isExpected {
			return nil, newUnexpectedAttributeError(node.name, attr)
		}
	}

	if node.hasChildren() {
		return nil, errNodeCannotHaveChildren
	}
	customName, hasCustomName := node.attributes[customNameAttrID]
	if !hasCustomName {
		return nil, newMissingArgumentError(node.name, customNameAttrID)
	}
	currentFormat, err := getCurrentFormat(node, formatFromParent, formats)
	if err != nil {
		return nil, err
	}
	args := CustomReceiverInitArgs{
		XmlCustomAttrs: dataCustomPrefixes,
	}

	if cfg != nil && cfg.CustomReceiverProducers != nil {
		if prod, ok := cfg.CustomReceiverProducers[customName]; ok {
			rec, err := prod(args)
			if err != nil {
				return nil, err
			}
			creceiver, err := NewCustomReceiverDispatcherByValue(currentFormat, rec, customName, args)
			if err != nil {
				return nil, err
			}
			err = rec.AfterParse(args)
			if err != nil {
				return nil, err
			}
			return creceiver, nil
		}
	}

	return NewCustomReceiverDispatcher(currentFormat, customName, args)
}

func createFilter(node *xmlNode, formatFromParent *formatter, formats map[string]*formatter, cfg *CfgParseParams) (interface{}, error) {
	err := checkUnexpectedAttribute(node, outputFormatID, filterLevelsAttrID)
	if err != nil {
		return nil, err
	}

	if !node.hasChildren() {
		return nil, errNodeMustHaveChildren
	}

	currentFormat, err := getCurrentFormat(node, formatFromParent, formats)
	if err != nil {
		return nil, err
	}

	levelsStr, isLevels := node.attributes[filterLevelsAttrID]
	if !isLevels {
		return nil, newMissingArgumentError(node.name, filterLevelsAttrID)
	}

	levels, err := parseLevels(levelsStr)
	if err != nil {
		return nil, err
	}

	receivers, err := createInnerReceivers(node, currentFormat, formats, cfg)
	if err != nil {
		return nil, err
	}

	return NewFilterDispatcher(currentFormat, receivers, levels...)
}

func createfileWriter(node *xmlNode, formatFromParent *formatter, formats map[string]*formatter, cfg *CfgParseParams) (interface{}, error) {
	err := checkUnexpectedAttribute(node, outputFormatID, pathID)
	if err != nil {
		return nil, err
	}

	if node.hasChildren() {
		return nil, errNodeCannotHaveChildren
	}

	currentFormat, err := getCurrentFormat(node, formatFromParent, formats)
	if err != nil {
		return nil, err
	}

	path, isPath := node.attributes[pathID]
	if !isPath {
		return nil, newMissingArgumentError(node.name, pathID)
	}

	fileWriter, err := NewFileWriter(path)
	if err != nil {
		return nil, err
	}

	return NewFormattedWriter(fileWriter, currentFormat)
}

// Creates new SMTP writer if encountered in the config file.
func createSMTPWriter(node *xmlNode, formatFromParent *formatter, formats map[string]*formatter, cfg *CfgParseParams) (interface{}, error) {
	err := checkUnexpectedAttribute(node, outputFormatID, senderaddressID, senderNameID, hostNameID, hostPortID, userNameID, userPassID, subjectID)
	if err != nil {
		return nil, err
	}
	// Node must have children.
	if !node.hasChildren() {
		return nil, errNodeMustHaveChildren
	}
	currentFormat, err := getCurrentFormat(node, formatFromParent, formats)
	if err != nil {
		return nil, err
	}
	senderAddress, ok := node.attributes[senderaddressID]
	if !ok {
		return nil, newMissingArgumentError(node.name, senderaddressID)
	}
	senderName, ok := node.attributes[senderNameID]
	if !ok {
		return nil, newMissingArgumentError(node.name, senderNameID)
	}
	// Process child nodes scanning for recipient email addresses and/or CA certificate paths.
	var recipientAddresses []string
	var caCertDirPaths []string
	var mailHeaders []string
	for _, childNode := range node.children {
		switch childNode.name {
		// Extract recipient address from child nodes.
		case recipientID:
			address, ok := childNode.attributes[addressID]
			if !ok {
				return nil, newMissingArgumentError(childNode.name, addressID)
			}
			recipientAddresses = append(recipientAddresses, address)
		// Extract CA certificate file path from child nodes.
		case cACertDirpathID:
			path, ok := childNode.attributes[pathID]
			if !ok {
				return nil, newMissingArgumentError(childNode.name, pathID)
			}
			caCertDirPaths = append(caCertDirPaths, path)

		// Extract email headers from child nodes.
		case mailHeaderID:
			headerName, ok := childNode.attributes[mailHeaderNameID]
			if !ok {
				return nil, newMissingArgumentError(childNode.name, mailHeaderNameID)
			}

			headerValue, ok := childNode.attributes[mailHeaderValueID]
			if !ok {
				return nil, newMissingArgumentError(childNode.name, mailHeaderValueID)
			}

			// Build header line
			mailHeaders = append(mailHeaders, fmt.Sprintf("%s: %s", headerName, headerValue))
		default:
			return nil, newUnexpectedChildElementError(childNode.name)
		}
	}
	hostName, ok := node.attributes[hostNameID]
	if !ok {
		return nil, newMissingArgumentError(node.name, hostNameID)
	}

	hostPort, ok := node.attributes[hostPortID]
	if !ok {
		return nil, newMissingArgumentError(node.name, hostPortID)
	}

	// Check if the string can really be converted into int.
	if _, err := strconv.Atoi(hostPort); err != nil {
		return nil, errors.New("invalid host port number")
	}

	userName, ok := node.attributes[userNameID]
	if !ok {
		return nil, newMissingArgumentError(node.name, userNameID)
	}

	userPass, ok := node.attributes[userPassID]
	if !ok {
		return nil, newMissingArgumentError(node.name, userPassID)
	}

	// subject is optionally set by configuration.
	// default value is defined by DefaultSubjectPhrase constant in the writers_smtpwriter.go
	var subjectPhrase = DefaultSubjectPhrase

	subject, ok := node.attributes[subjectID]
	if ok {
		subjectPhrase = subject
	}

	smtpWriter := NewSMTPWriter(
		senderAddress,
		senderName,
		recipientAddresses,
		hostName,
		hostPort,
		userName,
		userPass,
		caCertDirPaths,
		subjectPhrase,
		mailHeaders,
	)

	return NewFormattedWriter(smtpWriter, currentFormat)
}

func createConsoleWriter(node *xmlNode, formatFromParent *formatter, formats map[string]*formatter, cfg *CfgParseParams) (interface{}, error) {
	err := checkUnexpectedAttribute(node, outputFormatID)
	if err != nil {
		return nil, err
	}

	if node.hasChildren() {
		return nil, errNodeCannotHaveChildren
	}

	currentFormat, err := getCurrentFormat(node, formatFromParent, formats)
	if err != nil {
		return nil, err
	}

	consoleWriter, err := NewConsoleWriter()
	if err != nil {
		return nil, err
	}

	return NewFormattedWriter(consoleWriter, currentFormat)
}

func createconnWriter(node *xmlNode, formatFromParent *formatter, formats map[string]*formatter, cfg *CfgParseParams) (interface{}, error) {
	if node.hasChildren() {
		return nil, errNodeCannotHaveChildren
	}

	err := checkUnexpectedAttribute(node, outputFormatID, connWriterAddrAttr, connWriterNetAttr, connWriterReconnectOnMsgAttr, connWriterUseTLSAttr, connWriterInsecureSkipVerifyAttr)
	if err != nil {
		return nil, err
	}

	currentFormat, err := getCurrentFormat(node, formatFromParent, formats)
	if err != nil {
		return nil, err
	}

	addr, isAddr := node.attributes[connWriterAddrAttr]
	if !isAddr {
		return nil, newMissingArgumentError(node.name, connWriterAddrAttr)
	}

	net, isNet := node.attributes[connWriterNetAttr]
	if !isNet {
		return nil, newMissingArgumentError(node.name, connWriterNetAttr)
	}

	reconnectOnMsg := false
	reconnectOnMsgStr, isReconnectOnMsgStr := node.attributes[connWriterReconnectOnMsgAttr]
	if isReconnectOnMsgStr {
		if reconnectOnMsgStr == "true" {
			reconnectOnMsg = true
		} else if reconnectOnMsgStr == "false" {
			reconnectOnMsg = false
		} else {
			return nil, errors.New("node '" + node.name + "' has incorrect '" + connWriterReconnectOnMsgAttr + "' attribute value")
		}
	}

	useTLS := false
	useTLSStr, isUseTLSStr := node.attributes[connWriterUseTLSAttr]
	if isUseTLSStr {
		if useTLSStr == "true" {
			useTLS = true
		} else if useTLSStr == "false" {
			useTLS = false
		} else {
			return nil, errors.New("node '" + node.name + "' has incorrect '" + connWriterUseTLSAttr + "' attribute value")
		}
		if useTLS {
			insecureSkipVerify := false
			insecureSkipVerifyStr, isInsecureSkipVerify := node.attributes[connWriterInsecureSkipVerifyAttr]
			if isInsecureSkipVerify {
				if insecureSkipVerifyStr == "true" {
					insecureSkipVerify = true
				} else if insecureSkipVerifyStr == "false" {
					insecureSkipVerify = false
				} else {
					return nil, errors.New("node '" + node.name + "' has incorrect '" + connWriterInsecureSkipVerifyAttr + "' attribute value")
				}
			}
			config := tls.Config{InsecureSkipVerify: insecureSkipVerify}
			connWriter := newTLSWriter(net, addr, reconnectOnMsg, &config)
			return NewFormattedWriter(connWriter, currentFormat)
		}
	}

	connWriter := NewConnWriter(net, addr, reconnectOnMsg)

	return NewFormattedWriter(connWriter, currentFormat)
}

func createRollingFileWriter(node *xmlNode, formatFromParent *formatter, formats map[string]*formatter, cfg *CfgParseParams) (interface{}, error) {
	if node.hasChildren() {
		return nil, errNodeCannotHaveChildren
	}

	rollingTypeStr, isRollingType := node.attributes[rollingFileTypeAttr]
	if !isRollingType {
		return nil, newMissingArgumentError(node.name, rollingFileTypeAttr)
	}

	rollingType, ok := rollingTypeFromString(rollingTypeStr)
	if !ok {
		return nil, errors.New("unknown rolling file type: " + rollingTypeStr)
	}

	currentFormat, err := getCurrentFormat(node, formatFromParent, formats)
	if err != nil {
		return nil, err
	}

	path, isPath := node.attributes[rollingFilePathAttr]
	if !isPath {
		return nil, newMissingArgumentError(node.name, rollingFilePathAttr)
	}

	rollingArchiveStr, archiveAttrExists := node.attributes[rollingFileArchiveAttr]

	var rArchiveType rollingArchiveType
	var rArchivePath string
	var rArchiveExploded bool = false
	if !archiveAttrExists {
		rArchiveType = rollingArchiveNone
		rArchivePath = ""
	} else {
		rArchiveType, ok = rollingArchiveTypeFromString(rollingArchiveStr)
		if !ok {
			return nil, errors.New("unknown rolling archive type: " + rollingArchiveStr)
		}

		if rArchiveType == rollingArchiveNone {
			rArchivePath = ""
		} else {
			if rArchiveExplodedAttr, ok := node.attributes[rollingFileArchiveExplodedAttr]; ok {
				if rArchiveExploded, err = strconv.ParseBool(rArchiveExplodedAttr); err != nil {
					return nil, fmt.Errorf("archive exploded should be true or false, but was %v",
						rArchiveExploded)
				}
			}

			rArchivePath, ok = node.attributes[rollingFileArchivePathAttr]
			if ok {
				if rArchivePath == "" {
					return nil, fmt.Errorf("empty archive path is not supported")
				}
			} else {
				if rArchiveExploded {
					rArchivePath = rollingArchiveDefaultExplodedName

				} else {
					rArchivePath, err = rollingArchiveTypeDefaultName(rArchiveType, false)
					if err != nil {
						return nil, err
					}
				}
			}
		}
	}

	nameMode := rollingNameMode(rollingNameModePostfix)
	nameModeStr, ok := node.attributes[rollingFileNameModeAttr]
	if ok {
		mode, found := rollingNameModeFromString(nameModeStr)
		if !found {
			return nil, errors.New("unknown rolling filename mode: " + nameModeStr)
		} else {
			nameMode = mode
		}
	}

	if rollingType == rollingTypeSize {
		err := checkUnexpectedAttribute(node, outputFormatID, rollingFileTypeAttr, rollingFilePathAttr,
			rollingFileMaxSizeAttr, rollingFileMaxRollsAttr, rollingFileArchiveAttr,
			rollingFileArchivePathAttr, rollingFileArchiveExplodedAttr, rollingFileNameModeAttr)
		if err != nil {
			return nil, err
		}

		maxSizeStr, ok := node.attributes[rollingFileMaxSizeAttr]
		if !ok {
			return nil, newMissingArgumentError(node.name, rollingFileMaxSizeAttr)
		}

		maxSize, err := strconv.ParseInt(maxSizeStr, 10, 64)
		if err != nil {
			return nil, err
		}

		maxRolls := 0
		maxRollsStr, ok := node.attributes[rollingFileMaxRollsAttr]
		if ok {
			maxRolls, err = strconv.Atoi(maxRollsStr)
			if err != nil {
				return nil, err
			}
		}

		rollingWriter, err := NewRollingFileWriterSize(path, rArchiveType, rArchivePath, maxSize, maxRolls, nameMode, rArchiveExploded)
		if err != nil {
			return nil, err
		}

		return NewFormattedWriter(rollingWriter, currentFormat)

	} else if rollingType == rollingTypeTime {
		err := checkUnexpectedAttribute(node, outputFormatID, rollingFileTypeAttr, rollingFilePathAttr,
			rollingFileDataPatternAttr, rollingFileArchiveAttr, rollingFileMaxRollsAttr,
			rollingFileArchivePathAttr, rollingFileArchiveExplodedAttr, rollingFileNameModeAttr,
			rollingFileFullNameAttr)
		if err != nil {
			return nil, err
		}

		maxRolls := 0
		maxRollsStr, ok := node.attributes[rollingFileMaxRollsAttr]
		if ok {
			maxRolls, err = strconv.Atoi(maxRollsStr)
			if err != nil {
				return nil, err
			}
		}

		fullName := false
		fn, ok := node.attributes[rollingFileFullNameAttr]
		if ok {
			if fn == "true" {
				fullName = true
			} else if fn == "false" {
				fullName = false
			} else {
				return nil, errors.New("node '" + node.name + "' has incorrect '" + rollingFileFullNameAttr + "' attribute value")
			}
		}

		dataPattern, ok := node.attributes[rollingFileDataPatternAttr]
		if !ok {
			return nil, newMissingArgumentError(node.name, rollingFileDataPatternAttr)
		}

		rollingWriter, err := NewRollingFileWriterTime(path, rArchiveType, rArchivePath, maxRolls, dataPattern, nameMode, rArchiveExploded, fullName)
		if err != nil {
			return nil, err
		}

		return NewFormattedWriter(rollingWriter, currentFormat)
	}

	return nil, errors.New("incorrect rolling writer type " + rollingTypeStr)
}

func createbufferedWriter(node *xmlNode, formatFromParent *formatter, formats map[string]*formatter, cfg *CfgParseParams) (interface{}, error) {
	err := checkUnexpectedAttribute(node, outputFormatID, bufferedSizeAttr, bufferedFlushPeriodAttr)
	if err != nil {
		return nil, err
	}

	if !node.hasChildren() {
		return nil, errNodeMustHaveChildren
	}

	currentFormat, err := getCurrentFormat(node, formatFromParent, formats)
	if err != nil {
		return nil, err
	}

	sizeStr, isSize := node.attributes[bufferedSizeAttr]
	if !isSize {
		return nil, newMissingArgumentError(node.name, bufferedSizeAttr)
	}

	size, err := strconv.Atoi(sizeStr)
	if err != nil {
		return nil, err
	}

	flushPeriod := 0
	flushPeriodStr, isFlushPeriod := node.attributes[bufferedFlushPeriodAttr]
	if isFlushPeriod {
		flushPeriod, err = strconv.Atoi(flushPeriodStr)
		if err != nil {
			return nil, err
		}
	}

	// Inner writer couldn't have its own format, so we pass 'currentFormat' as its parent format
	receivers, err := createInnerReceivers(node, currentFormat, formats, cfg)
	if err != nil {
		return nil, err
	}

	formattedWriter, ok := receivers[0].(*formattedWriter)
	if !ok {
		return nil, errors.New("buffered writer's child is not writer")
	}

	// ... and then we check that it hasn't changed
	if formattedWriter.Format() != currentFormat {
		return nil, errors.New("inner writer cannot have his own format")
	}

	bufferedWriter, err := NewBufferedWriter(formattedWriter.Writer(), size, time.Duration(flushPeriod))
	if err != nil {
		return nil, err
	}

	return NewFormattedWriter(bufferedWriter, currentFormat)
}

// Returns an error if node has any attributes not listed in expectedAttrs.
func checkUnexpectedAttribute(node *xmlNode, expectedAttrs ...string) error {
	for attr := range node.attributes {
		isExpected := false
		for _, expected := range expectedAttrs {
			if attr == expected {
				isExpected = true
				break
			}
		}
		if !isExpected {
			return newUnexpectedAttributeError(node.name, attr)
		}
	}

	return nil
}

type expectedElementInfo struct {
	name      string
	mandatory bool
	multiple  bool
}

func optionalElement(name string) expectedElementInfo {
	return expectedElementInfo{name, false, false}
}
func mandatoryElement(name string) expectedElementInfo {
	return expectedElementInfo{name, true, false}
}
func multipleElements(name string) expectedElementInfo {
	return expectedElementInfo{name, false, true}
}
func multipleMandatoryElements(name string) expectedElementInfo {
	return expectedElementInfo{name, true, true}
}

func checkExpectedElements(node *xmlNode, elements ...expectedElementInfo) error {
	for _, element := range elements {
		count := 0
		for _, child := range node.children {
			if child.name == element.name {
				count++
			}
		}

		if count == 0 && element.mandatory {
			return errors.New(node.name + " does not have mandatory subnode - " + element.name)
		}
		if count > 1 && !element.multiple {
			return errors.New(node.name + " has more then one subnode - " + element.name)
		}
	}

	for _, child := range node.children {
		isExpected := false
		for _, element := range elements {
			if child.name == element.name {
				isExpected = true
			}
		}

		if !isExpected {
			return errors.New(node.name + " has unexpected child: " + child.name)
		}
	}

	return nil
}
