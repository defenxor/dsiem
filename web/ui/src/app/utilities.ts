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

export const sleep =  (ms: number) =>  new Promise(resolve => setTimeout(resolve, ms));

export const removeItemFromObjectArray = (array: object[], field: string, id: string) => {
  const idx = array.map(item => item[field]).indexOf(id);

  if (idx !== -1) {
    array.splice(idx, 1);
  }

};

export const parallelPromiseAllFlow = async (ids: any[], func: any): Promise<any[]> => {
  const promises = ids.map(id => func(id));
  const results = await Promise.all(promises);
  const finalResult = [];
  for (const result of results) {
    finalResult.push(await result);
  }
  return finalResult;
};

export const isEmptyOrUndefined = (v: any): boolean => v === '' || v === 0 || v === undefined;

export const url2obj = (url: string) => {
    const pattern = /^(?:([^:\/?#\s]+):\/{2})?(?:([^@\/?#\s]+)@)?([^\/?#\s]+)?(?:\/([^?#\s]*))?(?:[?]([^#\s]+))?\S*$/;
    const matches = url.match(pattern);

    return {
      protocol: matches[1],
      user: matches[2] !== undefined ? matches[2].split(':')[0] : undefined,
      password: matches[2] !== undefined ? matches[2].split(':')[1] : undefined,
      host: matches[3],
      hostname: matches[3] !== undefined ? matches[3].split(/:(?=\d+$)/)[0] : undefined,
      port: matches[3] !== undefined ? matches[3].split(/:(?=\d+$)/)[1] : undefined
    };
};
