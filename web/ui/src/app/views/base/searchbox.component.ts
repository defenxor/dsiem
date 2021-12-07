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
import { Component, Output, EventEmitter } from '@angular/core';

@Component({
  selector: 'app-search-box',
  templateUrl: 'searchbox.component.html',
})

export class SearchboxComponent {
  @Output() ready = new EventEmitter<string[]>();
  @Output() empty = new EventEmitter<boolean>();
  resultIDs: string[];
  /** @internal */
  validInput = true;
  isEmpty = false;
  alarmIDMinLength = 9;

  // emits signal when user stop typing and the search term is valid
  termReady($event: Event) {
    const s = ($event.target as HTMLInputElement).value;
    this.validInput = this.validateTerm(s);
    if (this.validInput) {
      this.ready.emit(this.resultIDs);
    }
  }

  // this emits signal when the searchbox is empty
  termEmpty() {
    this.isEmpty = true;
    this.empty.emit(true);
  }

  // convert comma separated IDs to array, return true only if
  // all array members length is >= alarmIDMinLength
  private validateTerm(term: string): boolean {
    const removeDup = (names) => names.filter((v, i) => names.indexOf(v) === i);
    this.resultIDs = term.replace(/\s+/g, '').split(',');
    this.resultIDs = removeDup(this.resultIDs);
    for (const id of this.resultIDs) {
      if (id.length < this.alarmIDMinLength) {
 return false;
}
    }
    return this.resultIDs.length > 0;
  }

}
