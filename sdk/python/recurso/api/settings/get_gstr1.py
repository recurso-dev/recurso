from http import HTTPStatus
from typing import Any

import httpx

from ... import errors
from ...client import AuthenticatedClient, Client
from ...models.error import Error
from ...models.get_gstr1_response_200 import GetGSTR1Response200
from ...types import UNSET, Response


def _get_kwargs(
    *,
    month: int,
    year: int,
) -> dict[str, Any]:

    params: dict[str, Any] = {}

    params["month"] = month

    params["year"] = year

    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}

    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/v1/india/gstr1",
        "params": params,
    }

    return _kwargs


def _parse_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Error | GetGSTR1Response200 | None:
    if response.status_code == 200:
        response_200 = GetGSTR1Response200.from_dict(response.json())

        return response_200

    if response.status_code == 400:
        response_400 = Error.from_dict(response.json())

        return response_400

    if response.status_code == 401:
        response_401 = Error.from_dict(response.json())

        return response_401

    if client.raise_on_unexpected_status:
        raise errors.UnexpectedStatus(response.status_code, response.content)
    else:
        return None


def _build_response(
    *, client: AuthenticatedClient | Client, response: httpx.Response
) -> Response[Error | GetGSTR1Response200]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
    month: int,
    year: int,
) -> Response[Error | GetGSTR1Response200]:
    """GSTR-1 outward-supply return for a tax period

     Assembles the GSTR-1 return from the tenant's finalized invoices and refund credit notes for the
    given month: B2B (registered buyers, invoice-level), B2CS (unregistered, rate-wise), CDNR (credit
    notes to registered buyers), an HSN rollup, and control totals. Returns the readable sections plus a
    `gov_schema` object in the GSTN GSTR-1 JSON shape (official field names, amounts in rupees) ready to
    validate against the government offline tool before filing.

    Args:
        month (int):
        year (int):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | GetGSTR1Response200]
    """

    kwargs = _get_kwargs(
        month=month,
        year=year,
    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)


def sync(
    *,
    client: AuthenticatedClient | Client,
    month: int,
    year: int,
) -> Error | GetGSTR1Response200 | None:
    """GSTR-1 outward-supply return for a tax period

     Assembles the GSTR-1 return from the tenant's finalized invoices and refund credit notes for the
    given month: B2B (registered buyers, invoice-level), B2CS (unregistered, rate-wise), CDNR (credit
    notes to registered buyers), an HSN rollup, and control totals. Returns the readable sections plus a
    `gov_schema` object in the GSTN GSTR-1 JSON shape (official field names, amounts in rupees) ready to
    validate against the government offline tool before filing.

    Args:
        month (int):
        year (int):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | GetGSTR1Response200
    """

    return sync_detailed(
        client=client,
        month=month,
        year=year,
    ).parsed


async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    month: int,
    year: int,
) -> Response[Error | GetGSTR1Response200]:
    """GSTR-1 outward-supply return for a tax period

     Assembles the GSTR-1 return from the tenant's finalized invoices and refund credit notes for the
    given month: B2B (registered buyers, invoice-level), B2CS (unregistered, rate-wise), CDNR (credit
    notes to registered buyers), an HSN rollup, and control totals. Returns the readable sections plus a
    `gov_schema` object in the GSTN GSTR-1 JSON shape (official field names, amounts in rupees) ready to
    validate against the government offline tool before filing.

    Args:
        month (int):
        year (int):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | GetGSTR1Response200]
    """

    kwargs = _get_kwargs(
        month=month,
        year=year,
    )

    response = await client.get_async_httpx_client().request(**kwargs)

    return _build_response(client=client, response=response)


async def asyncio(
    *,
    client: AuthenticatedClient | Client,
    month: int,
    year: int,
) -> Error | GetGSTR1Response200 | None:
    """GSTR-1 outward-supply return for a tax period

     Assembles the GSTR-1 return from the tenant's finalized invoices and refund credit notes for the
    given month: B2B (registered buyers, invoice-level), B2CS (unregistered, rate-wise), CDNR (credit
    notes to registered buyers), an HSN rollup, and control totals. Returns the readable sections plus a
    `gov_schema` object in the GSTN GSTR-1 JSON shape (official field names, amounts in rupees) ready to
    validate against the government offline tool before filing.

    Args:
        month (int):
        year (int):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | GetGSTR1Response200
    """

    return (
        await asyncio_detailed(
            client=client,
            month=month,
            year=year,
        )
    ).parsed
