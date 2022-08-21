package com.example.qrauth;

import androidx.activity.result.ActivityResultLauncher;
import androidx.appcompat.app.AlertDialog;
import androidx.appcompat.app.AppCompatActivity;

import android.content.Context;
import android.content.DialogInterface;
import android.os.AsyncTask;
import android.os.Bundle;
import android.os.Looper;
import android.widget.Button;
import android.widget.Toast;

import okhttp3.OkHttpClient;
import okhttp3.Request;
import okhttp3.Response;

import com.journeyapps.barcodescanner.ScanContract;
import com.journeyapps.barcodescanner.ScanOptions;

import java.io.IOException;

public class MainActivity extends AppCompatActivity {
    Button LaunchScan;
    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        setContentView(R.layout.activity_main);
        LaunchScan = findViewById(R.id.launchscan);
        LaunchScan.setOnClickListener(v->{LaunchScan_Click();});

    }

    private void LaunchScan_Click() {
        ScanOptions scanoptions= new ScanOptions();
        scanoptions.setOrientationLocked(true);
        scanoptions.setBeepEnabled(false);
        scanoptions.setCaptureActivity(capact.class);
        scanoptions.setDesiredBarcodeFormats(ScanOptions.QR_CODE);
        arl.launch(scanoptions);



    }
    ActivityResultLauncher<ScanOptions> arl = registerForActivityResult(new ScanContract(), result ->{
        if (result.getContents()!=null){
            AlertDialog.Builder builder = new AlertDialog.Builder(MainActivity.this );
            builder.setTitle("Do you want to connect?");


            builder.setItems(new CharSequence[]{"OK", "Cancel"}, new DialogInterface.OnClickListener() {
                @Override
                public void onClick(DialogInterface dialogInterface, int i) {
                    if (i==0){


                              Thread thread = new Thread(new Runnable() {
                                  @Override
                                  public void run() {
                                      Looper.prepare();
                                      final OkHttpClient client = new OkHttpClient();
                                      String token = "c81e8366-0d2c-42b3-8639-8cbc7373f71c";
                                      String url = "http://192.168.1.2:8080/authenticate?id=" + result.getContents() + "&token=" + token;
                                      Toast.makeText(MainActivity.this, url, Toast.LENGTH_LONG ).show();
                                      Request request = new Request.Builder()
                                              .url(url)
                                              .build();
                                      Response response = null;
                                      try {
                                          response = client.newCall(request).execute();
                                      } catch (IOException e) {
                                          e.printStackTrace();
                                      }
                                  }
                              });
                              thread.start();




                        dialogInterface.dismiss();
                        return;
                    }
                    dialogInterface.dismiss();

                }
            }).show();

        }
    } );



}
