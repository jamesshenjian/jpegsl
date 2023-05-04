# jpegsl
A simple golang parser for sequential lossless jpeg pixel data used in dicom ct files

Sample code:

	data, err := os.ReadFile(file)  //or get the pixel data value from dicom
 	if err != nil {
        //...
	}
    //call Decode to get an slice of integers
	decodedData := jpegsl.Decode(data)
    //cast slice elements based on tag PixelRepresentation (value 0: unsigned; 1:signed) and BitsAllocated to correct type
    //TODO