/** This file is part of the Plasmacash library.
*
* The plasmacash library is free software: you can redistribute it and/or modify
* it under the terms of the GNU Lesser General Public License as published by
* the Free Software Foundation, either version 3 of the License, or
* (at your option) any later version.
*
* The Plasmacash library is distributed in the hope that it will be useful,
* but WITHOUT ANY WARRANTY; without even the implied warranty of
* MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
* GNU Lesser General Public License for more details.
*
* You should have received a copy of the GNU Lesser General Public License
* along with the plasmacash library. If not, see <http://www.gnu.org/licenses/>.
*/


/**
 * @title  RLP Encoding Library for Solidity
 * @author Sam Mayo (sammayo888@gmail.com)
 * @dev Library for rlp encoding arbitrary bytes or lists.
 */

pragma solidity ^0.4.25;

library RLPEncode {
    uint8 constant STRING_SHORT_PREFIX = 0x80;
    uint8 constant STRING_LONG_PREFIX = 0xb7;
    uint8 constant LIST_SHORT_PREFIX = 0xc0;
    uint8 constant LIST_LONG_PREFIX = 0xf7;

    /// @dev Rlp encodes a bytes
    /// @param self The bytes to be encoded
    /// @return The rlp encoded bytes
	function encodeBytes(bytes memory self) internal pure returns (bytes) {
        bytes memory encoded;
        if(self.length == 1 && uint(self[0]) < 0x80) {
            encoded = new bytes(1);
            encoded = self;
        } else {
        	encoded = encode(self, STRING_SHORT_PREFIX, STRING_LONG_PREFIX);
		}
        return encoded;
    }

    /// @dev Rlp encodes a bytes[]. Note that the items in the bytes[] will not automatically be rlp encoded.
    /// @param self The bytes[] to be encoded
    /// @return The rlp encoded bytes[]
    function encodeList(bytes[] memory self) internal pure returns (bytes) {
    	bytes memory list = flatten(self);
	    bytes memory encoded = encode(list, LIST_SHORT_PREFIX, LIST_LONG_PREFIX);
        return encoded;
    }

    function encode(bytes memory self, uint8 prefix1, uint8 prefix2) private pure returns (bytes) {
    	uint selfPtr;
        assembly { selfPtr := add(self, 0x20) }

        bytes memory encoded;
        uint encodedPtr;

    	uint len = self.length;
        uint lenLen;
        uint i = 0x1;
	    while(len/i != 0) {
	        lenLen++;
	        i *= 0x100;
	    }

        if(len <= 55) {
		    encoded = new bytes(len+1);

            // length encoding byte
		    encoded[0] = byte(prefix1+len);

            // string/list contents
            assembly { encodedPtr := add(encoded, 0x21) }
            memcpy(encodedPtr, selfPtr, len);
        } else {
        	// 1 is the length of the length of the length
		    encoded = new bytes(1+lenLen+len);

            // length of the length encoding byte
		    encoded[0] = byte(prefix2+lenLen);

            // length bytes
		    for(i=1; i<=lenLen; i++) {
		        encoded[i] = byte((len/(0x100**(lenLen-i)))%0x100);
		    }

            // string/list contents
            assembly { encodedPtr := add(add(encoded, 0x21), lenLen) }
            memcpy(encodedPtr, selfPtr, len);
        }
        return encoded;
    }

    function flatten(bytes[] memory self) private pure returns (bytes) {
        if(self.length == 0) {
            return new bytes(0);
        }

        uint len;
    	for(uint i=0; i<self.length; i++) {
    		len += self[i].length;
        }

        bytes memory flattened = new bytes(len);
        uint flattenedPtr;
        assembly { flattenedPtr := add(flattened, 0x20) }

        for(i=0; i<self.length; i++) {
            bytes memory item = self[i];

            uint selfPtr;
            assembly { selfPtr := add(item, 0x20)}

            memcpy(flattenedPtr, selfPtr, item.length);
            flattenedPtr += self[i].length;
        }

        return flattened;
    }

    /// This function is from Nick Johnson's string utils library
    function memcpy(uint dest, uint src, uint len) private pure {
        // Copy word-length chunks while possible
        for(; len >= 32; len -= 32) {
            assembly {
                mstore(dest, mload(src))
            }
            dest += 32;
            src += 32;
        }

        // Copy remaining bytes
        uint mask = 256 ** (32 - len) - 1;
        assembly {
            let srcpart := and(mload(src), not(mask))
            let destpart := and(mload(dest), mask)
            mstore(dest, or(destpart, srcpart))
        }
    }
}
