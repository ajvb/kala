import {Component, OnInit, Inject} from '@angular/core';
import {HttpClient} from "@angular/common/http";

import {MatDialog, MatDialogRef, MAT_DIALOG_DATA, MatSnackBar} from '@angular/material';
import {DialogConfirmDialog} from "./confirm.component";
import {DialogJobStatsDialog} from "./job.stats.component";
import {
  Job, JobStat, ListJobs, ListJobsResponse, KalaStats, KalaStatResponse, RemoteType,
  JobStatResponse
} from "./interfaces";

// @todo features: run manually, disable, enable, modify

const OperationKalaMetrics = 0;
const OperationListJobs = 1;
const OperationCreate = 2;

@Component({
  selector: 'dialog-job-detail-dialog',
  templateUrl: 'dialog-job-detail-dialog.html',
})

export class DialogJobDetailDialog {
  hidden: boolean;
  isDisabling: boolean;
  isEnabling: boolean;

  constructor(
    public dialogRef: MatDialogRef<DialogJobDetailDialog>,
    public http: HttpClient,
    @Inject(MAT_DIALOG_DATA) public data: any,
    public dialog: MatDialog,
    public snackBar: MatSnackBar,
    ) { }

  onNoClick(): void {
    this.dialogRef.close();
  }

  enable(job: Job):void {
    let dialogRef = this.dialog.open(DialogConfirmDialog, {
      width: '50%',
      data: {title: "Confirm", desc: `Do you really want to ENABLE "${job.name}"?`},
    });

    dialogRef.afterClosed().subscribe(confirmed => {
      if (confirmed) {
        this.http.post(`http://localhost:8000/api/v1/job/enable/${job.id}/`, null).subscribe(res => {
          console.log(res);
          this.snackBar.open("Enabled", "", {duration: 2500});
          job.disabled = false;
        })
      }
    });
  }
  disable(job: Job): void {
    let dialogRef = this.dialog.open(DialogConfirmDialog, {
      width: '50%',
      data: {title: "Confirm", desc: `Do you really want to DISABLE "${job.name}"?`},
    });

    dialogRef.afterClosed().subscribe(confirmed => {
      if (confirmed) {
        this.http.post(`http://localhost:8000/api/v1/job/disable/${job.id}/`, null).subscribe(res => {
          console.log(res);
          job.disabled = true;
          this.snackBar.open("Disabled", "", {duration: 2500});
        })
      }
    });
  }

  modify(job: Job): void {

  }
  run(job: Job): void {
    let dialogRef = this.dialog.open(DialogConfirmDialog, {
      width: '50%',
      data: {title: "Confirm", desc: `Do you really want to RUN "${job.name}"?`},
    });

    dialogRef.afterClosed().subscribe(confirmed => {
      if (confirmed) {
        this.http.post(`http://localhost:8000/api/v1/job/start/${job.id}/`, null).subscribe(res => {
          console.log(res);
          this.snackBar.open("Started", "", {duration: 2500});
        })
      }
    });
  }
  showStats(job: Job): void  {
    this.http.get<JobStatResponse>(`http://localhost:8000/api/v1/job/stats/${job.id}/`).subscribe(res => {
      job.stats = res.job_stats;
      this.hidden = true;
      let dialogRef = this.dialog.open(DialogJobStatsDialog, {
        width: '70%',
        data: {job_name: job.name, stats: job.stats},
      });
      dialogRef.afterClosed().subscribe(() => {
        this.hidden = false
      })
    });
  }

  delete(job: Job):void {
    let dialogRef = this.dialog.open(DialogConfirmDialog, {
      width: '50%',
      data: {title: "Confirm", desc: `Do you really want to delete "${job.name}"?`},
    });

    dialogRef.afterClosed().subscribe(confirmed => {
      if (confirmed) {
        this.http.delete("http://localhost:8000/api/v1/job/" + job.id + "/").subscribe(() => {
          let index = this.data.jobs.indexOf(job)
          if (index > -1) {
            this.data.jobs.splice(index, 1)
            this.dialogRef.close()
          }
        })
      }
    });
  }

}

@Component({
  selector: 'app-root',
  templateUrl: './app.component.html',
  styleUrls: ['./app.component.css']
})


export class AppComponent implements OnInit {
  title = 'Kala';
  defaultOperation: number = OperationCreate;
  operations: {create: number, listJobs: number, kalaMetrics: number} = {
    create: OperationCreate,
    listJobs: OperationListJobs,
    kalaMetrics: OperationKalaMetrics,
  };

  group: {value: string} = {value: "listJobs"};

  KalaStat: KalaStats = {
    active_jobs: 0,
    disabled_jobs: 0,
    jobs: 0,
    error_count: 0,
    success_count: 0,
    next_run_at: "",
    last_attempted_run: "",
    created: "",
  };
  KalaJobs: Job[] = [];

  constructor(private http: HttpClient, public dialog: MatDialog) {}
  ngOnInit():void {
    switch (this.defaultOperation) {
      case OperationKalaMetrics:
        this.metrics()
        break;
      case OperationCreate:

        break;
      case OperationListJobs:
        this.listJobs()
        break;
    }
  }

  listJobs(): void {
    this.http.get<ListJobsResponse>("http://localhost:8000/api/v1/job/").subscribe(data => {
      for (let id in data.jobs) {
        const job = data.jobs[id]
        if (job.type == RemoteType) {
          job.remote_properties.headersStr = JSON.stringify(job.remote_properties.headers)
        }
        this.KalaJobs.push(job)
      }
    })
  }

  metrics(): void {
    this.http.get<KalaStatResponse>("http://localhost:8000/api/v1/stats/").subscribe(data => {
      this.KalaStat = data.Stats
    })
  }

  openDialog(job: Job): void {
    // this.http.get<JobDetailResponse>("http://localhost:8000/api/v1/job/"+id+"/").subscribe(res => {
    //
    //   console.log(res)
    //   let dialogRef = this.dialog.open(DialogJobDetailDialog, {
    //     width: '70%',
    //     data: res.job,
    //   });
    //
    // });
      let dialogRef = this.dialog.open(DialogJobDetailDialog, {
        width: '70%',
        data: {job: job, jobs: this.KalaJobs},
      });
      dialogRef.afterClosed().subscribe(result => {
        console.log('The dialog was closed');
      });
  }
}
