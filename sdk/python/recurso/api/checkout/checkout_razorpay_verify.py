from http import HTTPStatus
from typing import Any
from urllib.parse import quote
from uuid import UUID

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.checkout_razorpay_verify_body import CheckoutRazorpayVerifyBody
from ...models.checkout_razorpay_verify_response_200 import CheckoutRazorpayVerifyResponse200
from ...models.error import Error
from ...types import Response


def _get_kwargs(
    id: UUID,
    *,
    body: CheckoutRazorpayVerifyBody,
) -> dict[str, Any]:
    headers: dict[str, Any] = {}

    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/checkout/{id}/razorpay/verify".format(
            id=quote(str(id), safe=""),
        ),
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> CheckoutRazorpayVerifyResponse200 | Error | None:
    if response.status_code == 200:
        response_200 = CheckoutRazorpayVerifyResponse200.from_dict(response.json())

        return response_200

    if response.status_code == 400:
        response_400 = Error.from_dict(response.json())

        return response_400

    if response.status_code == 404:
        response_404 = Error.from_dict(response.json())

        return response_404

    if response.status_code == 503:
        response_503 = Error.from_dict(response.json())

        return response_503

    if client.raise_on_unexpected_status:
        raise errors.UnexpectedStatus(response.status_code, response.content)
    else:
        return None


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[CheckoutRazorpayVerifyResponse200 | Error]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    id: UUID,
    *,
    client: AuthenticatedClient | Client,
    body: CheckoutRazorpayVerifyBody,
) -> Response[CheckoutRazorpayVerifyResponse200 | Error]:
    """Verify a Razorpay checkout payment and settle the invoice

     Called by the hosted checkout after the Razorpay modal reports success. Verifies the HMAC signature,
    fetches the order from Razorpay and confirms its `notes.invoice_id` matches this invoice, then
    settles through the idempotent ledger path. The `payment.captured` webhook remains the authoritative
    backstop.

    Args:
        id (UUID):
        body (CheckoutRazorpayVerifyBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[CheckoutRazorpayVerifyResponse200 | Error]
    """

    kwargs = _get_kwargs(
        id=id,
        body=body,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    id: UUID,
    *,
    client: AuthenticatedClient | Client,
    body: CheckoutRazorpayVerifyBody,
) -> CheckoutRazorpayVerifyResponse200 | Error | None:
    """Verify a Razorpay checkout payment and settle the invoice

     Called by the hosted checkout after the Razorpay modal reports success. Verifies the HMAC signature,
    fetches the order from Razorpay and confirms its `notes.invoice_id` matches this invoice, then
    settles through the idempotent ledger path. The `payment.captured` webhook remains the authoritative
    backstop.

    Args:
        id (UUID):
        body (CheckoutRazorpayVerifyBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        CheckoutRazorpayVerifyResponse200 | Error
    """

    return sync_detailed(
        id=id,
        client=client,
        body=body,
    ).parsed


async def asyncio_detailed(
    id: UUID,
    *,
    client: AuthenticatedClient | Client,
    body: CheckoutRazorpayVerifyBody,
) -> Response[CheckoutRazorpayVerifyResponse200 | Error]:
    """Verify a Razorpay checkout payment and settle the invoice

     Called by the hosted checkout after the Razorpay modal reports success. Verifies the HMAC signature,
    fetches the order from Razorpay and confirms its `notes.invoice_id` matches this invoice, then
    settles through the idempotent ledger path. The `payment.captured` webhook remains the authoritative
    backstop.

    Args:
        id (UUID):
        body (CheckoutRazorpayVerifyBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[CheckoutRazorpayVerifyResponse200 | Error]
    """

    kwargs = _get_kwargs(
        id=id,
        body=body,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    id: UUID,
    *,
    client: AuthenticatedClient | Client,
    body: CheckoutRazorpayVerifyBody,
) -> CheckoutRazorpayVerifyResponse200 | Error | None:
    """Verify a Razorpay checkout payment and settle the invoice

     Called by the hosted checkout after the Razorpay modal reports success. Verifies the HMAC signature,
    fetches the order from Razorpay and confirms its `notes.invoice_id` matches this invoice, then
    settles through the idempotent ledger path. The `payment.captured` webhook remains the authoritative
    backstop.

    Args:
        id (UUID):
        body (CheckoutRazorpayVerifyBody):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        CheckoutRazorpayVerifyResponse200 | Error
    """

    return (
        await asyncio_detailed(
            id=id,
            client=client,
            body=body,
        )
    ).parsed
