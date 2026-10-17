import json
import logging
from google.protobuf import json_format
from google.protobuf.message import DecodeError

logger = logging.getLogger(__name__)

def make_deserializer(pb_class):
    def _deserialize(raw_bytes: bytes):
        if not raw_bytes:
            return pb_class()
        
        # 1. Try standard binary protobuf decoding first
        try:
            msg = pb_class()
            msg.ParseFromString(raw_bytes)
            return msg
        except Exception:
            pass
        
        # 2. Try JSON decoding (used by Go Node Agent)
        try:
            json_str = raw_bytes.decode('utf-8', errors='replace')
            msg = pb_class()
            json_format.Parse(json_str, msg, ignore_unknown_fields=True)
            return msg
        except Exception as e:
            logger.error(f"[gRPC Codec] Failed to deserialize {pb_class.__name__}: {e}")
            raise DecodeError(f"Cannot deserialize {pb_class.__name__}: {e}")

    return _deserialize

def flexible_serializer(msg):
    if msg is None:
        return b""
    try:
        d = json_format.MessageToDict(
            msg,
            preserving_proto_field_name=True,
            always_print_fields_with_no_presence=False
        )

        def _cast_numbers(obj):
            if isinstance(obj, dict):
                return {k: _cast_numbers(v) for k, v in obj.items()}
            elif isinstance(obj, list):
                return [_cast_numbers(v) for v in obj]
            elif isinstance(obj, str) and obj.isdigit():
                return int(obj)
            return obj

        clean_dict = _cast_numbers(d)
        return json.dumps(clean_dict).encode('utf-8')
    except Exception as e:
        logger.warning(f"[gRPC Codec] Failed flexible_serializer: {e}, falling back to binary")
        return msg.SerializeToString()
