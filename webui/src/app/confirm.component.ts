import {Component, Inject} from '@angular/core';

import {MatDialogRef, MAT_DIALOG_DATA} from '@angular/material';

interface DialogConfirmData {
  title: string
  desc: string
}

@Component({
  selector: 'dialog-confirm-dialog',
  templateUrl: 'dialog-confirm-dialog.html',
})

export class DialogConfirmDialog {

  constructor(
    public dialogRef: MatDialogRef<DialogConfirmDialog>,
    @Inject(MAT_DIALOG_DATA) public data: DialogConfirmData) { }

  confirm():void {
    this.dialogRef.close(true)
  }

  cancel(): void {
    this.dialogRef.close(false)
  }
}
