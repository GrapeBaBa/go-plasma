/** Copyright 2018 Wolk Inc.
* This file is part of the Plasmacash library.
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
 * @title  Merkle
 * @author Michael Chung (michael@wolk.com)
 * @dev merkleProof implementation.
 */

 pragma solidity ^0.4.25;

 library Merkle {

     function getRoot(bytes32 leaf, uint256 index, bytes proof) internal pure returns (bytes32) {
         bytes32 proofElement;
         bytes32 computedHash = leaf;

         for (uint256 i = 32; i <= proof.length; i += 32) {
             assembly {
                 proofElement := mload(add(proof, i))
             }
             if (index % 2 == 0) {
                 computedHash = keccak256(abi.encodePacked(computedHash, proofElement));
             } else {
                 computedHash = keccak256(abi.encodePacked(proofElement, computedHash));
             }
             index = index / 2;
         }
         return computedHash;
     }
 }
