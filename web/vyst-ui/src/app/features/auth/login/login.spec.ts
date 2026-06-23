import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { of } from 'rxjs';

import { LoginComponent } from './login';
import { AuthService } from '../../../core/services/auth';
import { CaptchaService } from '../../../core/captcha/captcha.service';

describe('LoginComponent', () => {
  let component: LoginComponent;
  let fixture: ComponentFixture<LoginComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [LoginComponent],
      providers: [
        provideRouter([]),
        {
          provide: AuthService,
          useValue: {
            login: () => of({}),
            verify2FA: () => of({}),
            beginPasskeyLogin: () => of({ publicKey: { challenge: '', allowCredentials: [] } }),
            finishPasskeyLogin: () => of({}),
            tempToken: { set: () => undefined },
            requires2FA: { set: () => undefined },
          },
        },
        {
          provide: CaptchaService,
          useValue: {
            getConfig: () => of({ enabled: false, site_key: '' }),
            loadScript: () => Promise.resolve(),
            render: () => undefined,
            remove: () => undefined,
            reset: () => undefined,
            clearCache: () => undefined,
          },
        },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(LoginComponent);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
