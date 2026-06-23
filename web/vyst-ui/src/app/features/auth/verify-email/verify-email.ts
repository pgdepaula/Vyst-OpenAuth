import { Component, OnInit, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, Router, RouterModule } from '@angular/router';
import { HttpClient } from '@angular/common/http';
import { catchError, of } from 'rxjs';

@Component({
  selector: 'app-verify-email',
  standalone: true,
  imports: [CommonModule, RouterModule],
  templateUrl: './verify-email.html',
  styleUrl: './verify-email.css'
})
export class VerifyEmailComponent implements OnInit {
  private route = inject(ActivatedRoute);
  private router = inject(Router);
  private http = inject(HttpClient);

  status = signal<'verifying' | 'success' | 'error'>('verifying');
  errorMessage = signal<string>('');

  ngOnInit() {
    const token = this.route.snapshot.queryParamMap.get('token');
    if (!token) {
      this.status.set('error');
      this.errorMessage.set('Invalid verification link.');
      return;
    }

    this.verifyEmail(token);
  }

  verifyEmail(token: string) {
    this.http.get('/api/auth/verify-email', { params: { token } })
      .pipe(
        catchError(err => {
          this.status.set('error');
          this.errorMessage.set(err.error?.error || 'Verification failed. Please try again.');
          return of(null);
        })
      )
      .subscribe(res => {
        if (res) {
          this.status.set('success');
          setTimeout(() => {
            this.router.navigate(['/login']);
          }, 3000);
        }
      });
  }
}
