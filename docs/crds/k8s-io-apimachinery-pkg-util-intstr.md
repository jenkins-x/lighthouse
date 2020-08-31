# Package k8s.io/apimachinery/pkg/util/intstr

- [IntOrString](#IntOrString)


## IntOrString

IntOrString is a type that can hold an int32 or a string.  When used in<br />JSON or YAML marshalling and unmarshalling, it produces or consumes the<br />inner type.  This allows you to have, for example, a JSON field that can<br />accept a name or number.<br />TODO: Rename to Int32OrString<br /><br />+protobuf=true<br />+protobuf.options.(gogoproto.goproto_stringer)=false<br />+k8s:openapi-gen=true




