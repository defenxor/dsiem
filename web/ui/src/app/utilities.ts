/*
Copyright (c) 2019 PT Defender Nusa Semesta and contributors, All rights reserved.

This file is part of Dsiem.

Dsiem is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation version 3 of the License.

Dsiem is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with Dsiem. If not, see <https:www.gnu.org/licenses/>.
*/

export function sleep (ms) {
  return new Promise(resolve => setTimeout(resolve, ms));
}

export function removeItemFromObjectArray(array: object[], field: string, id: string) {
  const removeIndex = array
    .map(item => item[field])
    .indexOf(id);
  if (removeIndex !== -1) {
    array.splice(removeIndex, 1);
  }
}

export async function parallelPromiseAllFlow(IDs: any[], func): Promise<any[]> {
  const promises = IDs.map(id => func(id));
  const results = await Promise.all(promises);
  const finalResult = [];
  for (const result of results) {
    finalResult.push(await result);
  }
  return finalResult;
}
