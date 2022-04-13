# Package k8s.io/apimachinery/pkg/runtime

- [RawExtension](#RawExtension)


## RawExtension

RawExtension is used to hold extensions in external versions.<br /><br />To use this, make a field which has RawExtension as its type in your external, versioned<br />struct, and Object in your internal struct. You also need to register your<br />various plugin types.<br /><br />// Internal package:<br />type MyAPIObject struct {<br />	runtime.TypeMeta `json:",inline"`<br />	MyPlugin runtime.Object `json:"myPlugin"`<br />}<br />type PluginA struct {<br />	AOption string `json:"aOption"`<br />}<br /><br />// External package:<br />type MyAPIObject struct {<br />	runtime.TypeMeta `json:",inline"`<br />	MyPlugin runtime.RawExtension `json:"myPlugin"`<br />}<br />type PluginA struct {<br />	AOption string `json:"aOption"`<br />}<br /><br />// On the wire, the JSON will look something like this:<br />{<br />	"kind":"MyAPIObject",<br />	"apiVersion":"v1",<br />	"myPlugin": {<br />		"kind":"PluginA",<br />		"aOption":"foo",<br />	},<br />}<br /><br />So what happens? Decode first uses json or yaml to unmarshal the serialized data into<br />your external MyAPIObject. That causes the raw JSON to be stored, but not unpacked.<br />The next step is to copy (using pkg/conversion) into the internal struct. The runtime<br />package's DefaultScheme has conversion functions installed which will unpack the<br />JSON stored in RawExtension, turning it into the correct object type, and storing it<br />in the Object. (TODO: In the case where the object is of an unknown type, a<br />runtime.Unknown object will be created and stored.)<br /><br />+k8s:deepcopy-gen=true<br />+protobuf=true<br />+k8s:openapi-gen=true




