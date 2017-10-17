import {Component, Inject} from '@angular/core';
import {MatDialog, MatDialogRef, MAT_DIALOG_DATA} from '@angular/material';

interface JobStat {
  run_at: string
  number_of_retries: number
  output: string
  success: boolean
  execution_duration: number
}

@Component({
  selector: 'dialog-job-stats-dialog',
  templateUrl: 'dialog-job-stats-dialog.html',
})

export class DialogJobStatsDialog {
  stats: JobStat[] = [];
  pageSize: number = 10;
  pageSizeOptions: number[] = [5, 10, 25, 50, 100];

  constructor(
    public dialogRef: MatDialogRef<DialogJobStatsDialog>,
    @Inject(MAT_DIALOG_DATA) public data: {job_name: string, stats: JobStat[]}) {
    this._page(0, this.pageSize)
  }

  page(event):void {
    this._page(event.pageIndex, event.pageSize)
  }

  _page(index, size):void {
    console.log(index, size)
    let startIndex = index * size;
    this.stats.splice(0, this.stats.length)
    for (let i = 0 ; i < size; i++) {
      if (!(i in this.data.stats)) {
        break;
      }
      this.stats.push(this.data.stats[i + startIndex])
    }
  }
}
